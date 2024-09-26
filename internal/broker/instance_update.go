package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/assuredworkloads"

	"github.com/kyma-incubator/compass/components/director/pkg/jsonschema"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
	"github.com/kyma-project/kyma-environment-broker/internal/k8s"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
)

type ContextUpdateHandler interface {
	Handle(instance *internal.Instance, newCtx internal.ERSContext) (bool, error)
}

type UpdateEndpoint struct {
	config Config
	log    logrus.FieldLogger

	instanceStorage                         storage.Instances
	runtimeStates                           storage.RuntimeStates
	contextUpdateHandler                    ContextUpdateHandler
	brokerURL                               string
	processingEnabled                       bool
	subaccountMovementEnabled               bool
	updateCustomResouresLabelsOnAccountMove bool

	operationStorage storage.Operations

	updatingQueue Queue

	plansConfig  PlansConfig
	planDefaults PlanDefaults

	dashboardConfig dashboard.Config
	kcBuilder       kubeconfig.KcBuilder

	convergedCloudRegionsProvider ConvergedCloudRegionProvider

	kcpClient client.Client
}

func NewUpdate(cfg Config,
	instanceStorage storage.Instances,
	runtimeStates storage.RuntimeStates,
	operationStorage storage.Operations,
	ctxUpdateHandler ContextUpdateHandler,
	processingEnabled bool,
	subaccountMovementEnabled bool,
	updateCustomResouresLabelsOnAccountMove bool,
	queue Queue,
	plansConfig PlansConfig,
	planDefaults PlanDefaults,
	log logrus.FieldLogger,
	dashboardConfig dashboard.Config,
	kcBuilder kubeconfig.KcBuilder,
	convergedCloudRegionsProvider ConvergedCloudRegionProvider,
	kcpClient client.Client,
) *UpdateEndpoint {
	return &UpdateEndpoint{
		config:                                  cfg,
		log:                                     log.WithField("service", "UpdateEndpoint"),
		instanceStorage:                         instanceStorage,
		runtimeStates:                           runtimeStates,
		operationStorage:                        operationStorage,
		contextUpdateHandler:                    ctxUpdateHandler,
		processingEnabled:                       processingEnabled,
		subaccountMovementEnabled:               subaccountMovementEnabled,
		updateCustomResouresLabelsOnAccountMove: updateCustomResouresLabelsOnAccountMove,
		updatingQueue:                           queue,
		plansConfig:                             plansConfig,
		planDefaults:                            planDefaults,
		dashboardConfig:                         dashboardConfig,
		kcBuilder:                               kcBuilder,
		convergedCloudRegionsProvider:           convergedCloudRegionsProvider,
		kcpClient:                               kcpClient,
	}
}

// Update modifies an existing service instance
//
//	PATCH /v2/service_instances/{instance_id}
func (b *UpdateEndpoint) Update(_ context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("Updating instanceID: %s", instanceID)
	logger.Infof("Updating asyncAllowed: %v", asyncAllowed)
	logger.Infof("Parameters: '%s'", string(details.RawParameters))
	instance, err := b.instanceStorage.GetByID(instanceID)
	if err != nil && dberr.IsNotFound(err) {
		logger.Errorf("unable to get instance: %s", err.Error())
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, fmt.Sprintf("could not execute update for instanceID %s", instanceID))
	} else if err != nil {
		logger.Errorf("unable to get instance: %s", err.Error())
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to get instance")
	}
	logger.Infof("Plan ID/Name: %s/%s", instance.ServicePlanID, PlanNamesMapping[instance.ServicePlanID])
	var ersContext internal.ERSContext
	err = json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Errorf("unable to decode context: %s", err.Error())
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to unmarshal context")
	}
	logger.Infof("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active))
	logger.Infof("Received context: %s", marshallRawContext(hideSensitiveDataFromRawContext(details.RawContext)))
	// validation of incoming input
	if err := b.validateWithJsonSchemaValidator(details, instance); err != nil {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "validation failed")
	}

	if instance.IsExpired() {
		if b.config.AllowUpdateExpiredInstanceWithContext && ersContext.GlobalAccountID != "" {
			return domain.UpdateServiceSpec{}, nil
		}
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("cannot update an expired instance"), http.StatusBadRequest, "")
	}
	lastProvisioningOperation, err := b.operationStorage.GetProvisioningOperationByInstanceID(instance.InstanceID)
	if err != nil {
		logger.Errorf("cannot fetch provisioning lastProvisioningOperation for instance with ID: %s : %s", instance.InstanceID, err.Error())
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to process the update")
	}
	if lastProvisioningOperation.State == domain.Failed {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("Unable to process an update of a failed instance"), http.StatusUnprocessableEntity, "")
	}

	lastDeprovisioningOperation, err := b.operationStorage.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		logger.Errorf("cannot fetch deprovisioning for instance with ID: %s : %s", instance.InstanceID, err.Error())
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to process the update")
	}
	if err == nil {
		if !lastDeprovisioningOperation.Temporary {
			// it is not a suspension, but real deprovisioning
			logger.Warnf("Cannot process update, the instance has started deprovisioning process (operationID=%s)", lastDeprovisioningOperation.Operation.ID)
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("Unable to process an update of a deprovisioned instance"), http.StatusUnprocessableEntity, "")
		}
	}

	dashboardURL := instance.DashboardURL
	if b.dashboardConfig.LandscapeURL != "" {
		dashboardURL = fmt.Sprintf("%s/?kubeconfigID=%s", b.dashboardConfig.LandscapeURL, instanceID)
		instance.DashboardURL = dashboardURL
	}

	if b.processingEnabled {
		instance, suspendStatusChange, err := b.processContext(instance, details, lastProvisioningOperation, logger)
		if err != nil {
			return domain.UpdateServiceSpec{}, err
		}

		// NOTE: KEB currently can't process update parameters in one call along with context update
		// this block makes it that KEB ignores any parameters updates if context update changed suspension state
		if !suspendStatusChange && !instance.IsExpired() {
			return b.processUpdateParameters(instance, details, lastProvisioningOperation, asyncAllowed, ersContext, logger)
		}
	}
	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  dashboardURL,
		OperationData: "",
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel, b.kcBuilder),
		},
	}, nil
}

func (b *UpdateEndpoint) validateWithJsonSchemaValidator(details domain.UpdateDetails, instance *internal.Instance) error {
	if len(details.RawParameters) > 0 {
		planValidator, err := b.getJsonSchemaValidator(instance.Provider, instance.ServicePlanID, instance.Parameters.PlatformRegion)
		if err != nil {
			return fmt.Errorf("while creating plan validator: %w", err)
		}
		result, err := planValidator.ValidateString(string(details.RawParameters))
		if err != nil {
			return fmt.Errorf("while executing JSON schema validator: %w", err)
		}
		if !result.Valid {
			return fmt.Errorf("while validating update parameters: %w", result.Error)
		}
	}
	return nil
}

func shouldUpdate(instance *internal.Instance, details domain.UpdateDetails, ersContext internal.ERSContext) bool {
	if len(details.RawParameters) != 0 {
		return true
	}
	return ersContext.ERSUpdate()
}

func (b *UpdateEndpoint) processUpdateParameters(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, asyncAllowed bool, ersContext internal.ERSContext, logger logrus.FieldLogger) (domain.UpdateServiceSpec, error) {
	if !shouldUpdate(instance, details, ersContext) {
		logger.Debugf("Parameters not provided, skipping processing update parameters")
		return domain.UpdateServiceSpec{
			IsAsync:       false,
			DashboardURL:  instance.DashboardURL,
			OperationData: "",
			Metadata: domain.InstanceMetadata{
				Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel, b.kcBuilder),
			},
		}, nil
	}
	// asyncAllowed needed, see https://github.com/openservicebrokerapi/servicebroker/blob/v2.16/spec.md#updating-a-service-instance
	if !asyncAllowed {
		return domain.UpdateServiceSpec{}, apiresponses.ErrAsyncRequired
	}
	var params internal.UpdatingParametersDTO
	if len(details.RawParameters) != 0 {
		err := json.Unmarshal(details.RawParameters, &params)
		if err != nil {
			logger.Errorf("unable to unmarshal parameters: %s", err.Error())
			return domain.UpdateServiceSpec{}, fmt.Errorf("unable to unmarshal parameters")
		}
		logger.Debugf("Updating with params: %+v", params)
	}

	if params.OIDC.IsProvided() {
		if err := params.OIDC.Validate(); err != nil {
			logger.Errorf("invalid OIDC parameters: %s", err.Error())
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}
	}

	operationID := uuid.New().String()
	logger = logger.WithField("operationID", operationID)

	logger.Debugf("creating update operation %v", params)
	operation := internal.NewUpdateOperation(operationID, instance, params)
	planID := instance.Parameters.PlanID
	if len(details.PlanID) != 0 {
		planID = details.PlanID
	}
	defaults, err := b.planDefaults(planID, instance.Provider, &instance.Provider)
	if err != nil {
		logger.Errorf("unable to obtain plan defaults: %s", err.Error())
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to obtain plan defaults")
	}
	var autoscalerMin, autoscalerMax int
	if defaults.GardenerConfig != nil {
		p := defaults.GardenerConfig
		autoscalerMin, autoscalerMax = p.AutoScalerMin, p.AutoScalerMax
	}
	if err := operation.ProvisioningParameters.Parameters.AutoScalerParameters.Validate(autoscalerMin, autoscalerMax); err != nil {
		logger.Errorf("invalid autoscaler parameters: %s", err.Error())
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
	}
	err = b.operationStorage.InsertOperation(operation)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}

	var updateStorage []string
	if params.OIDC.IsProvided() {
		instance.Parameters.Parameters.OIDC = params.OIDC
		updateStorage = append(updateStorage, "OIDC")
	}

	if len(params.RuntimeAdministrators) != 0 {
		newAdministrators := make([]string, 0, len(params.RuntimeAdministrators))
		newAdministrators = append(newAdministrators, params.RuntimeAdministrators...)
		instance.Parameters.Parameters.RuntimeAdministrators = newAdministrators
		updateStorage = append(updateStorage, "Runtime Administrators")
	}

	if params.UpdateAutoScaler(&instance.Parameters.Parameters) {
		updateStorage = append(updateStorage, "Auto Scaler parameters")
	}
	if params.MachineType != nil && *params.MachineType != "" {
		instance.Parameters.Parameters.MachineType = params.MachineType
	}
	if len(updateStorage) > 0 {
		if err := wait.PollImmediate(500*time.Millisecond, 2*time.Second, func() (bool, error) {
			instance, err = b.instanceStorage.Update(*instance)
			if err != nil {
				params := strings.Join(updateStorage, ", ")
				logger.Warnf("unable to update instance with new %v (%s), retrying", params, err.Error())
				return false, nil
			}
			return true, nil
		}); err != nil {
			response := apiresponses.NewFailureResponse(fmt.Errorf("Update operation failed"), http.StatusInternalServerError, err.Error())
			return domain.UpdateServiceSpec{}, response
		}
	}
	logger.Debugf("Adding update operation to the processing queue")
	b.updatingQueue.Add(operationID)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		DashboardURL:  instance.DashboardURL,
		OperationData: operation.ID,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*lastProvisioningOperation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel, b.kcBuilder),
		},
	}, nil
}

func (b *UpdateEndpoint) processContext(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, logger logrus.FieldLogger) (*internal.Instance, bool, error) {
	var ersContext internal.ERSContext
	err := json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Errorf("unable to decode context: %s", err.Error())
		return nil, false, fmt.Errorf("unable to unmarshal context")
	}
	logger.Infof("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active))

	lastOp, err := b.operationStorage.GetLastOperation(instance.InstanceID)
	if err != nil {
		logger.Errorf("unable to get last operation: %s", err.Error())
		return nil, false, fmt.Errorf("failed to process ERS context")
	}

	// todo: remove the code below when we are sure the ERSContext contains required values.
	// This code is done because the PATCH request contains only some of fields and that requests made the ERS context empty in the past.
	existingSMOperatorCredentials := instance.Parameters.ErsContext.SMOperatorCredentials
	instance.Parameters.ErsContext = lastProvisioningOperation.ProvisioningParameters.ErsContext
	// but do not change existing SM operator credentials
	instance.Parameters.ErsContext.SMOperatorCredentials = existingSMOperatorCredentials
	instance.Parameters.ErsContext.Active, err = b.extractActiveValue(instance.InstanceID, *lastProvisioningOperation)
	if err != nil {
		return nil, false, fmt.Errorf("unable to process the update")
	}
	instance.Parameters.ErsContext = internal.InheritMissingERSContext(instance.Parameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
	instance.Parameters.ErsContext = internal.UpdateInstanceERSContext(instance.Parameters.ErsContext, ersContext)

	changed, err := b.contextUpdateHandler.Handle(instance, ersContext)
	if err != nil {
		logger.Errorf("processing context updated failed: %s", err.Error())
		return nil, changed, fmt.Errorf("unable to process the update")
	}

	//  copy the Active flag if set
	if ersContext.Active != nil {
		instance.Parameters.ErsContext.Active = ersContext.Active
	}

	needUpdateCustomResources := false
	if b.subaccountMovementEnabled && (instance.GlobalAccountID != ersContext.GlobalAccountID && ersContext.GlobalAccountID != "") {
		if instance.SubscriptionGlobalAccountID == "" {
			instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
		}
		instance.GlobalAccountID = ersContext.GlobalAccountID
		needUpdateCustomResources = true
	}

	newInstance, err := b.instanceStorage.Update(*instance)
	if err != nil {
		logger.Errorf("processing context updated failed: %s", err.Error())
		return nil, changed, fmt.Errorf("unable to process the update")
	} else if b.updateCustomResouresLabelsOnAccountMove && needUpdateCustomResources {
		logger.Errorf("flag: %t", b.updateCustomResouresLabelsOnAccountMove)
		// update labels on related CRs, but only if account movement was successfully persisted and kept in database
		err = b.updateLabels(newInstance.RuntimeID, newInstance.GlobalAccountID)
		if err != nil {
			// silent error by design for now
			logger.Errorf("unable to update global account label on CRs while doing account move: %s", err.Error())
			response := apiresponses.NewFailureResponse(fmt.Errorf("Update CR failed"), http.StatusInternalServerError, err.Error())
			return newInstance, changed, response
		}
	}

	return newInstance, changed, nil
}

func (b *UpdateEndpoint) extractActiveValue(id string, provisioning internal.ProvisioningOperation) (*bool, error) {
	deprovisioning, dErr := b.operationStorage.GetDeprovisioningOperationByInstanceID(id)
	if dErr != nil && !dberr.IsNotFound(dErr) {
		b.log.Errorf("Unable to get deprovisioning operation for the instance %s to check the active flag: %s", id, dErr.Error())
		return nil, dErr
	}
	// there was no any deprovisioning in the past (any suspension)
	if deprovisioning == nil {
		return ptr.Bool(true), nil
	}

	return ptr.Bool(deprovisioning.CreatedAt.Before(provisioning.CreatedAt)), nil
}

func (b *UpdateEndpoint) getJsonSchemaValidator(provider internal.CloudProvider, planID string, platformRegion string) (JSONSchemaValidator, error) {
	// shootAndSeedSameRegion is never enabled for update
	b.log.Printf("region is: %s", platformRegion)
	plans := Plans(b.plansConfig, provider, b.config.IncludeAdditionalParamsInSchema, euaccess.IsEURestrictedAccess(platformRegion), b.config.UseSmallerMachineTypes, false, b.convergedCloudRegionsProvider.GetRegions(platformRegion), assuredworkloads.IsKSA(platformRegion))
	plan := plans[planID]
	schema := string(Marshal(plan.Schemas.Instance.Update.Parameters))

	return jsonschema.NewValidatorFromStringSchema(schema)
}

func (b *UpdateEndpoint) updateLabels(id, newGlobalAccountId string) error {
	kymaErr := b.updateCrLabel(id, k8s.KymaCr, newGlobalAccountId)
	gardenerClusterErr := b.updateCrLabel(id, k8s.GardenerClusterCr, newGlobalAccountId)
	runtimeErr := b.updateCrLabel(id, k8s.RuntimeCr, newGlobalAccountId)
	err := errors.Join(kymaErr, gardenerClusterErr, runtimeErr)
	return err
}

func (b *UpdateEndpoint) updateCrLabel(id, crName, newGlobalAccountId string) error {
	gvk, err := k8s.GvkByName(crName)
	if err != nil {
		return fmt.Errorf("while getting gvk for name: %s: %s", crName, err.Error())
	}

	var k8sObject unstructured.Unstructured
	k8sObject.SetGroupVersionKind(gvk)
	err = b.kcpClient.Get(context.Background(), types.NamespacedName{Namespace: KymaNamespace, Name: id}, &k8sObject)
	if err != nil {
		return fmt.Errorf("while getting k8s object of type %s from kcp cluster for instance %s, due to: %s", crName, id, err.Error())
	}

	err = k8s.AddOrOverrideMetadata(&k8sObject, k8s.GlobalAccountIdLabel, newGlobalAccountId)
	if err != nil {
		return fmt.Errorf("while adding or overriding label (new=%s) for k8s object %s %s, because: %s", newGlobalAccountId, id, crName, err.Error())
	}

	err = b.kcpClient.Update(context.Background(), &k8sObject)
	if err != nil {
		return fmt.Errorf("while updating k8s object %s %s, because: %s", id, crName, err.Error())
	}

	return nil
}
