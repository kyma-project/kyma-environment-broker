package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/additionalproperties"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/validator"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const planChangeMessage = "Plan change"

type ContextUpdateHandler interface {
	Handle(instance *internal.Instance, newCtx internal.ERSContext) (bool, error)
}

type UpdateEndpoint struct {
	config Config
	log    *slog.Logger

	instanceStorage                          storage.Instances
	contextUpdateHandler                     ContextUpdateHandler
	processingEnabled                        bool
	subaccountMovementEnabled                bool
	updateCustomResourcesLabelsOnAccountMove bool

	operationStorage storage.Operations
	actionStorage    storage.Actions

	updatingQueue Queue

	plansConfig PlansConfig

	dashboardConfig dashboard.Config
	kcBuilder       kubeconfig.KcBuilder

	kcpClient                   client.Client
	valuesProvider              ValuesProvider
	infrastructureManagerConfig InfrastructureManager

	schemaService    *SchemaService
	providerSpec     *configuration.ProviderSpec
	planSpec         *configuration.PlanSpecifications
	quotaClient      QuotaClient
	quotaWhitelist   whitelist.Set
	rulesService     *rules.RulesService
	gardenerClient   *gardener.Client
	awsClientFactory aws.ClientFactory

	useCredentialsBindings         bool
	syncEmptyUpdateResponseEnabled bool
}

func NewUpdate(cfg Config,
	db storage.BrokerStorage,
	ctxUpdateHandler ContextUpdateHandler,
	processingEnabled bool,
	subaccountMovementEnabled bool,
	updateCustomResourcesLabelsOnAccountMove bool,
	queue Queue,
	plansConfig PlansConfig,
	valuesProvider ValuesProvider,
	log *slog.Logger,
	dashboardConfig dashboard.Config,
	kcBuilder kubeconfig.KcBuilder,
	kcpClient client.Client,
	providerSpec *configuration.ProviderSpec,
	planSpec *configuration.PlanSpecifications,
	imConfig InfrastructureManager,
	schemaService *SchemaService,
	quotaClient QuotaClient,
	quotaWhitelist whitelist.Set,
	rulesService *rules.RulesService,
	gardenerClient *gardener.Client,
	awsClientFactory aws.ClientFactory,
) *UpdateEndpoint {
	return &UpdateEndpoint{
		config:                                   cfg,
		log:                                      log.With("service", "UpdateEndpoint"),
		instanceStorage:                          db.Instances(),
		contextUpdateHandler:                     ctxUpdateHandler,
		processingEnabled:                        processingEnabled,
		subaccountMovementEnabled:                subaccountMovementEnabled,
		updateCustomResourcesLabelsOnAccountMove: updateCustomResourcesLabelsOnAccountMove,
		operationStorage:                         db.Operations(),
		actionStorage:                            db.Actions(),
		updatingQueue:                            queue,
		plansConfig:                              plansConfig,
		dashboardConfig:                          dashboardConfig,
		kcBuilder:                                kcBuilder,
		kcpClient:                                kcpClient,
		valuesProvider:                           valuesProvider,
		infrastructureManagerConfig:              imConfig,
		schemaService:                            schemaService,
		providerSpec:                             providerSpec,
		planSpec:                                 planSpec,
		quotaClient:                              quotaClient,
		quotaWhitelist:                           quotaWhitelist,
		rulesService:                             rulesService,
		gardenerClient:                           gardenerClient,
		awsClientFactory:                         awsClientFactory,
		syncEmptyUpdateResponseEnabled:           cfg.SyncEmptyUpdateResponseEnabled,
	}
}

// Update modifies an existing service instance
//
//	PATCH /v2/service_instances/{instance_id}
func (b *UpdateEndpoint) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.With("instanceID", instanceID)
	logger.Info(fmt.Sprintf("Updating instanceID: %s", instanceID))
	logger.Info(fmt.Sprintf("Updating asyncAllowed: %v", asyncAllowed))
	logger.Info(fmt.Sprintf("Parameters: '%s'", string(details.RawParameters)))
	logger.Info(fmt.Sprintf("Plan ID: '%s'", details.PlanID))

	response, err := b.update(ctx, instanceID, details, asyncAllowed)
	if dberr.IsConflict(err) {
		logger.Warn("Update conflict, retrying")
		response, err = b.update(ctx, instanceID, details, asyncAllowed)
	}

	return response, err
}

func (b *UpdateEndpoint) update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	logger := b.log.With("instanceID", instanceID)
	instance, err := b.instanceStorage.GetByID(instanceID)
	if err != nil && dberr.IsNotFound(err) {
		logger.Error(fmt.Sprintf("unable to get instance: %s", err.Error()))
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, fmt.Sprintf("could not execute update for instanceID %s", instanceID))
	} else if err != nil {
		logger.Error(fmt.Sprintf("unable to get instance: %s", err.Error()))
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to get instance")
	}
	logger.Info(fmt.Sprintf("Plan ID/Name: %s/%s", instance.ServicePlanID, AvailablePlans.GetPlanNameOrEmpty(PlanIDType(instance.ServicePlanID))))
	var ersContext internal.ERSContext
	err = json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to decode context: %s", err.Error()))
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to unmarshal context")
	}
	logger.Info(fmt.Sprintf("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active)))
	logger.Info(fmt.Sprintf("Received context: %s", marshallRawContext(hideSensitiveDataFromRawContext(details.RawContext))))
	if b.config.MonitorAdditionalProperties {
		b.monitorAdditionalProperties(instanceID, ersContext, details.RawParameters)
	}
	// validation of incoming input
	if err := b.validateWithJsonSchemaValidator(details, instance); err != nil {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "validation failed")
	}

	if instance.IsExpired() {
		if ersContext.GlobalAccountID != "" {
			return domain.UpdateServiceSpec{}, nil
		}
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("cannot update an expired instance"), http.StatusBadRequest, "")
	}
	lastProvisioningOperation, err := b.operationStorage.GetProvisioningOperationByInstanceID(instance.InstanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot fetch provisioning lastProvisioningOperation for instance with ID: %s : %s", instance.InstanceID, err.Error()))
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to process the update")
	}
	if lastProvisioningOperation.State == domain.Failed {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("Unable to process an update of a failed instance"), http.StatusUnprocessableEntity, "")
	}

	lastDeprovisioningOperation, err := b.operationStorage.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		logger.Error(fmt.Sprintf("cannot fetch deprovisioning for instance with ID: %s : %s", instance.InstanceID, err.Error()))
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to process the update")
	}
	if err == nil {
		if !lastDeprovisioningOperation.Temporary {
			// it is not a suspension, but real deprovisioning
			logger.Warn(fmt.Sprintf("Cannot process update, the instance has started deprovisioning process (operationID=%s)", lastDeprovisioningOperation.ID))
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("Unable to process an update of a deprovisioned instance"), http.StatusUnprocessableEntity, "")
		}
	}

	dashboardURL := instance.DashboardURL
	if b.dashboardConfig.LandscapeURL != "" {
		dashboardURL = fmt.Sprintf("%s/?kubeconfigID=%s", b.dashboardConfig.LandscapeURL, instanceID)
		instance.DashboardURL = dashboardURL
	}

	if b.processingEnabled {
		previousInstance := *instance
		instance, suspendStatusChange, err := b.processContext(instance, details, lastProvisioningOperation, logger)
		if err != nil {
			return domain.UpdateServiceSpec{}, err
		}

		// NOTE: KEB currently can't process update parameters in one call along with context update
		// this block makes it that KEB ignores any parameters updates if context update changed suspension state
		if !suspendStatusChange && !instance.IsExpired() {
			return b.processUpdateParameters(ctx, &previousInstance, instance, details, lastProvisioningOperation, asyncAllowed, ersContext, logger)
		}
	}
	return domain.UpdateServiceSpec{
		IsAsync:       false,
		DashboardURL:  dashboard.ProvideURL(instance, lastProvisioningOperation),
		OperationData: "",
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*instance, b.config.URL, b.kcBuilder),
		},
	}, nil
}

func (b *UpdateEndpoint) validateWithJsonSchemaValidator(details domain.UpdateDetails, instance *internal.Instance) error {
	if len(details.RawParameters) > 0 {
		planValidator, err := b.getJsonSchemaValidator(instance.Provider, instance.ServicePlanID, instance.Parameters.PlatformRegion)
		if err != nil {
			return fmt.Errorf("while creating plan validator: %w", err)
		}
		var rawParameters any
		if err = json.Unmarshal(details.RawParameters, &rawParameters); err != nil {
			return fmt.Errorf("while unmarshaling raw parameters: %w", err)
		}
		if err = planValidator.Validate(rawParameters); err != nil {
			return fmt.Errorf("while validating update parameters: %s", validator.FormatError(err))
		}
	}
	return nil
}

func shouldUpdate(instance *internal.Instance, details domain.UpdateDetails, ersContext internal.ERSContext) bool {
	if len(details.RawParameters) != 0 {
		return true
	}
	if details.PlanID != instance.ServicePlanID {
		return true
	}
	return ersContext.ERSUpdate()
}

func (b *UpdateEndpoint) processUpdateParameters(ctx context.Context, previousInstance, instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, asyncAllowed bool, ersContext internal.ERSContext, logger *slog.Logger) (domain.UpdateServiceSpec, error) {
	if !shouldUpdate(instance, details, ersContext) {
		logger.Debug("Parameters not provided, skipping processing update parameters")
		return domain.UpdateServiceSpec{
			IsAsync:       false,
			DashboardURL:  dashboard.ProvideURL(instance, lastProvisioningOperation),
			OperationData: "",
			Metadata: domain.InstanceMetadata{
				Labels: ResponseLabels(*instance, b.config.URL, b.kcBuilder),
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
			logger.Error(fmt.Sprintf("unable to unmarshal parameters: %s", err.Error()))
			return domain.UpdateServiceSpec{}, fmt.Errorf("unable to unmarshal parameters")
		}
		logger.Debug(fmt.Sprintf("Updating with params: %+v", params))
	}
	// TODO: remove once we implemented proper filtering of parameters - removing parameters that are not supported by the plan
	if details.PlanID == TrialPlanID {
		params.MachineType = nil
		params.AutoScalerMin = nil
		params.AutoScalerMax = nil
	}

	providerValues, err := b.valuesProvider.ValuesForPlanAndParameters(instance.Parameters)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to obtain dummyProvider values: %s", err.Error()))
		return domain.UpdateServiceSpec{}, fmt.Errorf("unable to process the request")
	}

	regionsSupportingMachine, err := b.providerSpec.RegionSupportingMachine(providerValues.ProviderType)
	if err != nil {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
	}
	if !regionsSupportingMachine.IsSupported(valueOfPtr(instance.Parameters.Parameters.Region), valueOfPtr(params.MachineType)) {
		message := fmt.Sprintf(
			"In the region %s, the machine type %s is not available, it is supported in the %v",
			valueOfPtr(instance.Parameters.Parameters.Region),
			valueOfPtr(params.MachineType),
			strings.Join(regionsSupportingMachine.SupportedRegions(valueOfPtr(params.MachineType)), ", "),
		)
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusBadRequest, message)
	}

	discoveredZones := make(map[string]int)
	if b.providerSpec.ZonesDiscovery(pkg.CloudProviderFromString(providerValues.ProviderType)) {
		if params.MachineType != nil {
			discoveredZones[*params.MachineType] = 0
		}

		for _, additionalWorkerNodePool := range params.AdditionalWorkerNodePools {
			discoveredZones[additionalWorkerNodePool.MachineType] = 0
		}

		// todo: simplify it, remove "if" when all KCP instances are migrated to use credentials bindings
		var awsClient aws.Client
		if b.useCredentialsBindings {
			awsClient, err = newAWSClientUsingCredentialsBinding(ctx, logger, b.rulesService, b.gardenerClient, b.awsClientFactory, instance.Parameters, providerValues)
		} else {
			awsClient, err = newAWSClient(ctx, logger, b.rulesService, b.gardenerClient, b.awsClientFactory, instance.Parameters, providerValues)
		}
		if err != nil {
			logger.Error(fmt.Sprintf("unable to create AWS client: %s", err))
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New(FailedToValidateZonesMsg), http.StatusBadRequest, FailedToValidateZonesMsg)
		}

		for machineType := range discoveredZones {
			zonesCount, err := awsClient.AvailableZonesCount(ctx, machineType)
			if err != nil {
				logger.Error(fmt.Sprintf("unable to get available zones: %s", err))
				return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(errors.New(FailedToValidateZonesMsg), http.StatusBadRequest, FailedToValidateZonesMsg)
			}
			discoveredZones[machineType] = zonesCount
		}

		if params.MachineType != nil {
			if discoveredZones[*params.MachineType] < providerValues.ZonesCount {
				message := fmt.Sprintf("In the %s, the %s machine type is not available in %v zones.", providerValues.Region, *params.MachineType, providerValues.ZonesCount)
				return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusUnprocessableEntity, message)
			}
		}
	}

	if params.OIDC.IsProvided() {
		if err := params.OIDC.Validate(instance.Parameters.Parameters.OIDC); err != nil {
			logger.Error(fmt.Sprintf("invalid OIDC parameters: %s", err.Error()))
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
		}
	}

	operationID := uuid.New().String()
	logger = logger.With("operationID", operationID)

	logger.Debug(fmt.Sprintf("creating update operation %v", params))
	operation := internal.NewUpdateOperation(operationID, instance, params)
	operation.ProviderValues = &providerValues

	if err := operation.ProvisioningParameters.Parameters.AutoScalerParameters.Validate(providerValues.DefaultAutoScalerMin, providerValues.DefaultAutoScalerMax); err != nil {
		logger.Error(fmt.Sprintf("invalid autoscaler parameters: %s", err.Error()))
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
	}

	if params.AdditionalWorkerNodePools != nil {
		if !supportsAdditionalWorkerNodePools(details.PlanID) {
			message := fmt.Sprintf("additional worker node pools are not supported for plan ID: %s", details.PlanID)
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusBadRequest, message)
		}

		if !AreNamesUnique(params.AdditionalWorkerNodePools) {
			message := "names of additional worker node pools must be unique"
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusBadRequest, message)
		}

		if IsExternalLicenseType(ersContext) {
			if err := checkGPUMachinesUsage(params.AdditionalWorkerNodePools); err != nil {
				return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
			}
		}

		if err := checkUnsupportedMachines(regionsSupportingMachine, valueOfPtr(instance.Parameters.Parameters.Region), params.AdditionalWorkerNodePools); err != nil {
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
		}

		if err := checkAutoScalerConfiguration(params.AdditionalWorkerNodePools); err != nil {
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
		}

		if err := checkHAZonesUnchanged(instance.Parameters.Parameters.AdditionalWorkerNodePools, params.AdditionalWorkerNodePools); err != nil {
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
		}

		if err := checkAvailableZones(
			logger,
			regionsSupportingMachine,
			params.AdditionalWorkerNodePools,
			providerValues.Region,
			details.PlanID,
			b.providerSpec.ZonesDiscovery(pkg.CloudProviderFromString(providerValues.ProviderType)),
			discoveredZones,
		); err != nil {
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
		}

		multiError := pkg.MachineTypeMultiError{}
		for _, additionalWorkerNodePool := range params.AdditionalWorkerNodePools {
			planName := AvailablePlans.GetPlanNameOrEmpty(PlanIDType(details.PlanID))
			if err := additionalWorkerNodePool.ValidateMachineTypeChange(instance.Parameters.Parameters.AdditionalWorkerNodePools, b.planSpec.RegularMachines(planName)); err != nil {
				multiError.Append(err)
			}
		}
		if multiError.IsError() {
			return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(&multiError, http.StatusBadRequest, multiError.Error())
		}
	}

	err = validateIngressFiltering(operation.ProvisioningParameters, params.IngressFiltering, b.infrastructureManagerConfig.IngressFilteringPlans, logger)
	if err != nil {
		return domain.UpdateServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
	}

	updateStorage, err := b.updateInstanceAndOperationParameters(instance, &params, &operation, details, ersContext, logger)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}
	if len(updateStorage) > 0 {
		instance, err = b.instanceStorage.Update(*instance)
		if err != nil {
			params := strings.Join(updateStorage, ", ")
			logger.Warn(fmt.Sprintf("unable to update instance with new %v (%s)", params, err.Error()))
			return domain.UpdateServiceSpec{}, err
		}
		if slices.Contains(updateStorage, planChangeMessage) {
			oldPlan := AvailablePlans.GetPlanNameOrEmpty(PlanIDType(previousInstance.ServicePlanID))
			newPlan := AvailablePlans.GetPlanNameOrEmpty(PlanIDType(details.PlanID))
			message := fmt.Sprintf("Plan updated from %s (PlanID: %s) to %s (PlanID: %s).", oldPlan, previousInstance.ServicePlanID, newPlan, details.PlanID)
			if err := b.actionStorage.InsertAction(
				pkg.PlanUpdateActionType,
				instance.InstanceID,
				message,
				previousInstance.ServicePlanID,
				details.PlanID,
			); err != nil {
				logger.Error(fmt.Sprintf("while inserting action %q with message %s for instance ID %s: %v", pkg.PlanUpdateActionType, message, instance.InstanceID, err))
			}
		}
	}

	if skipProcessing, lastOperation := b.shouldSkipNewOperation(previousInstance, instance, logger); skipProcessing {
		return b.responseWithoutNewOperation(instance, lastOperation, logger)
	}
	logger.Debug("Creating update operation in the database")
	err = b.operationStorage.InsertOperation(operation)
	if err != nil {
		return domain.UpdateServiceSpec{}, err
	}
	logger.Debug("Adding update operation to the processing queue")
	b.updatingQueue.Add(operationID)

	return domain.UpdateServiceSpec{
		IsAsync:       true,
		DashboardURL:  dashboard.ProvideURL(instance, lastProvisioningOperation),
		OperationData: operation.ID,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*instance, b.config.URL, b.kcBuilder),
		},
	}, nil
}

func (b *UpdateEndpoint) shouldSkipNewOperation(previousInstance, currentInstance *internal.Instance, logger *slog.Logger) (bool, *internal.Operation) {
	if !b.syncEmptyUpdateResponseEnabled {
		return false, nil
	}
	// do not compare "Active" field
	previousInstance.Parameters.ErsContext.Active = currentInstance.Parameters.ErsContext.Active

	if !previousInstance.Parameters.IsEqual(currentInstance.Parameters) {
		logger.Info("Parameters changed, cannot skip new operation")
		if !reflect.DeepEqual(previousInstance.Parameters.ErsContext, currentInstance.Parameters.ErsContext) {
			logger.Info("Context changed")
		}
		if !reflect.DeepEqual(previousInstance.Parameters.Parameters, currentInstance.Parameters.Parameters) {
			logger.Info("Parameters changed")
		}
		return false, nil
	}
	if previousInstance.ServicePlanID != currentInstance.ServicePlanID {
		logger.Info("Plan changed, cannot skip new operation")
		return false, nil
	}
	if previousInstance.GlobalAccountID != currentInstance.GlobalAccountID {
		logger.Info("GlobalAccountID changed, cannot skip new operation")
		return false, nil
	}
	lastOperation, err := b.operationStorage.GetLastOperationWithAllStates(currentInstance.InstanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to get last operation: %s, cannot skip new operation", err.Error()))
		return false, nil
	}

	// if the last operation did not succeed, we cannot respond synchronously
	// Last change could fail, we cannot accept and response OK for next update which do the same.
	if lastOperation.State == domain.Failed {
		logger.Info(fmt.Sprintf("Last operation %s %s failed, cannot skip new operation", lastOperation.Type, lastOperation.ID))
		return false, lastOperation
	}

	return true, lastOperation
}

// UseCredentialsBindings indicates whether to use credentials bindings when creating AWS clients, it is a deprecated func and will be removed in future releases
// when all KCP instances are migrated to use credentials bindings
func (b *UpdateEndpoint) UseCredentialsBindings() {
	b.useCredentialsBindings = true
}

func (b *UpdateEndpoint) processContext(instance *internal.Instance, details domain.UpdateDetails, lastProvisioningOperation *internal.ProvisioningOperation, logger *slog.Logger) (*internal.Instance, bool, error) {
	var ersContext internal.ERSContext
	err := json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to decode context: %s", err.Error()))
		return nil, false, fmt.Errorf("unable to unmarshal context")
	}
	logger.Info(fmt.Sprintf("Global account ID: %s active: %s", instance.GlobalAccountID, ptr.BoolAsString(ersContext.Active)))

	lastOp, err := b.operationStorage.GetLastOperation(instance.InstanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to get last operation: %s", err.Error()))
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
		logger.Error(fmt.Sprintf("processing context updated failed: %s", err.Error()))
		return nil, changed, fmt.Errorf("unable to process the update")
	}

	//  copy the Active flag if set
	if ersContext.Active != nil {
		instance.Parameters.ErsContext.Active = ersContext.Active
	}

	needUpdateCustomResources := false
	if b.subaccountMovementEnabled && (instance.GlobalAccountID != ersContext.GlobalAccountID && ersContext.GlobalAccountID != "") {
		message := fmt.Sprintf("Subaccount %s moved from Global Account %s to %s.", ersContext.SubAccountID, instance.GlobalAccountID, ersContext.GlobalAccountID)
		if err := b.actionStorage.InsertAction(
			pkg.SubaccountMovementActionType,
			instance.InstanceID,
			message,
			instance.GlobalAccountID,
			ersContext.GlobalAccountID,
		); err != nil {
			logger.Error(fmt.Sprintf("while inserting action %q with message %s for instance ID %s: %v", pkg.SubaccountMovementActionType, message, instance.InstanceID, err))
		}
		if instance.SubscriptionGlobalAccountID == "" {
			instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
		}
		instance.GlobalAccountID = ersContext.GlobalAccountID
		needUpdateCustomResources = true
		logger.Info(fmt.Sprintf("Global account ID changed to: %s. need update labels", instance.GlobalAccountID))
	}

	newInstance, err := b.instanceStorage.Update(*instance)
	if err != nil {
		logger.Error(fmt.Sprintf("instance updated failed: %s", err.Error()))
		return nil, changed, err
	} else if b.updateCustomResourcesLabelsOnAccountMove && needUpdateCustomResources {
		logger.Info("updating labels on related CRs")
		// update labels on related CRs, but only if account movement was successfully persisted and kept in database
		labeler := NewLabeler(b.kcpClient)
		err = labeler.UpdateLabels(newInstance.RuntimeID, newInstance.GlobalAccountID)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to update global account label on CRs while doing account move: %s", err.Error()))
			response := apiresponses.NewFailureResponse(fmt.Errorf("Update CRs label failed"), http.StatusInternalServerError, err.Error())
			return newInstance, changed, response
		}
		logger.Info("labels updated")
	}

	return newInstance, changed, nil
}

func (b *UpdateEndpoint) extractActiveValue(id string, provisioning internal.ProvisioningOperation) (*bool, error) {
	deprovisioning, dErr := b.operationStorage.GetDeprovisioningOperationByInstanceID(id)
	if dErr != nil && !dberr.IsNotFound(dErr) {
		b.log.Error(fmt.Sprintf("Unable to get deprovisioning operation for the instance %s to check the active flag: %s", id, dErr.Error()))
		return nil, dErr
	}
	// there was no any deprovisioning in the past (any suspension)
	if deprovisioning == nil {
		return ptr.Bool(true), nil
	}

	return ptr.Bool(deprovisioning.CreatedAt.Before(provisioning.CreatedAt)), nil
}

func (b *UpdateEndpoint) getJsonSchemaValidator(provider pkg.CloudProvider, planID string, platformRegion string) (*jsonschema.Schema, error) {
	// colocateControlPlane is never enabled for update
	b.log.Info(fmt.Sprintf("region is: %s", platformRegion))

	plans := b.schemaService.Plans(b.plansConfig, platformRegion, provider)
	plan := plans[planID]

	return validator.NewFromSchema(plan.Schemas.Instance.Update.Parameters)
}

func (b *UpdateEndpoint) monitorAdditionalProperties(instanceID string, ersContext internal.ERSContext, rawParameters json.RawMessage) {
	var parameters internal.UpdatingParametersDTO
	decoder := json.NewDecoder(bytes.NewReader(rawParameters))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parameters); err == nil {
		return
	}
	if err := insertRequest(instanceID, filepath.Join(b.config.AdditionalPropertiesPath, additionalproperties.UpdateRequestsFileName), ersContext, rawParameters); err != nil {
		b.log.Error(fmt.Sprintf("failed to save update request with additional properties: %v", err))
	}
}

func (b *UpdateEndpoint) responseWithoutNewOperation(instance *internal.Instance, lastOperation *internal.Operation, logger *slog.Logger) (domain.UpdateServiceSpec, error) {
	logger.Info("parameters did not change, skipping creation of an operation")
	instance.EmptyUpdates++
	_, err := b.instanceStorage.Update(*instance)
	if err != nil {
		// storing EmptyUpdates failed, but we can still return OK response
		logger.Error(fmt.Sprintf("unable to update instance: %s", err.Error()))
	}
	if lastOperation != nil && lastOperation.Type == internal.OperationTypeUpdate && lastOperation.State != domain.Succeeded {
		logger.Info(fmt.Sprintf("last operation (%s) was an update and did not finished yet, returning the existing operation ID", lastOperation.ID))
		return domain.UpdateServiceSpec{
			IsAsync:       true,
			DashboardURL:  instance.DashboardURL,
			OperationData: lastOperation.ID,
			Metadata: domain.InstanceMetadata{
				Labels: ResponseLabels(*instance, b.config.URL, b.kcBuilder),
			},
		}, nil
	}
	return domain.UpdateServiceSpec{
		IsAsync: false,
	}, nil
}

func (b *UpdateEndpoint) updateInstanceAndOperationParameters(instance *internal.Instance, params *internal.UpdatingParametersDTO, operation *internal.Operation, details domain.UpdateDetails, ersContext internal.ERSContext, logger *slog.Logger) ([]string, error) {
	var updateStorage []string
	if details.PlanID != "" && details.PlanID != instance.ServicePlanID {
		logger.Info(fmt.Sprintf("Plan change requested: %s -> %s", instance.ServicePlanID, details.PlanID))
		if b.config.EnablePlanUpgrades && b.planSpec.IsUpgradableBetween(AvailablePlans.GetPlanNameOrEmpty(PlanIDType(instance.ServicePlanID)), AvailablePlans.GetPlanNameOrEmpty(PlanIDType(details.PlanID))) {
			if b.config.CheckQuotaLimit && whitelist.IsNotWhitelisted(ersContext.SubAccountID, b.quotaWhitelist) {
				if err := validateQuotaLimit(b.instanceStorage, b.quotaClient, ersContext.SubAccountID, details.PlanID, true); err != nil {
					return nil, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
				}
			}
			logger.Info("Plan change accepted.")
			operation.UpdatedPlanID = details.PlanID
			operation.ProvisioningParameters.PlanID = details.PlanID
			instance.Parameters.PlanID = details.PlanID
			instance.ServicePlanID = details.PlanID
			instance.ServicePlanName = AvailablePlans.GetPlanNameOrEmpty(PlanIDType(details.PlanID))
			updateStorage = append(updateStorage, planChangeMessage)
		} else {
			logger.Info("Plan change not allowed.")
			sourcePlanName := AvailablePlans.GetPlanNameOrEmpty(PlanIDType(instance.ServicePlanID))
			targetPlanName := AvailablePlans.GetPlanNameOrEmpty(PlanIDType(details.PlanID))
			return nil, apiresponses.NewFailureResponse(
				fmt.Errorf("plan upgrade from %s (planID: %s) to %s (planID: %s) is not allowed", sourcePlanName, instance.ServicePlanID, targetPlanName, details.PlanID),
				http.StatusBadRequest,
				fmt.Sprintf("plan upgrade from %s (planID: %s) to %s (planID: %s) is not allowed", sourcePlanName, instance.ServicePlanID, targetPlanName, details.PlanID),
			)
		}
	}

	if params.OIDC.IsProvided() {
		if params.OIDC.List != nil || (params.OIDC.OIDCConfigDTO != nil && !params.OIDC.OIDCConfigDTO.IsEmpty()) {
			instance.Parameters.Parameters.OIDC = params.OIDC
			updateStorage = append(updateStorage, "OIDC")
		}
	}

	if params.IngressFiltering != nil {
		instance.Parameters.Parameters.IngressFiltering = params.IngressFiltering
		updateStorage = append(updateStorage, "Ingress Filtering")
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
		updateStorage = append(updateStorage, "Machine type")
	}

	if supportsAdditionalWorkerNodePools(details.PlanID) && params.AdditionalWorkerNodePools != nil {
		newAdditionalWorkerNodePools := make([]pkg.AdditionalWorkerNodePool, 0, len(params.AdditionalWorkerNodePools))
		newAdditionalWorkerNodePools = append(newAdditionalWorkerNodePools, params.AdditionalWorkerNodePools...)
		instance.Parameters.Parameters.AdditionalWorkerNodePools = newAdditionalWorkerNodePools
		updateStorage = append(updateStorage, "Additional Worker Node Pools")
	}

	if params.Name != nil && *params.Name != "" {
		instance.Parameters.Parameters.Name = *params.Name
		updateStorage = append(updateStorage, "Cluster Name")
	}

	return updateStorage, nil
}

func checkHAZonesUnchanged(currentAdditionalWorkerNodePools, newAdditionalWorkerNodePools []pkg.AdditionalWorkerNodePool) error {
	var poolsWithChangedHAZones []string
	for _, additionalWorkerNodePool := range newAdditionalWorkerNodePools {
		if !additionalWorkerNodePool.ValidateHAZonesUnchanged(currentAdditionalWorkerNodePools) {
			poolsWithChangedHAZones = append(poolsWithChangedHAZones, additionalWorkerNodePool.Name)
		}
	}

	if len(poolsWithChangedHAZones) == 0 {
		return nil
	}

	message := fmt.Sprintf("HA zones setting is permanent and cannot be changed for additional worker node pools: %s.", strings.Join(poolsWithChangedHAZones, ", "))
	return fmt.Errorf("%s", message)
}
