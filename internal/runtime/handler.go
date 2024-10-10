package runtime

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"golang.org/x/exp/slices"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/common/pagination"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
)

const numberOfUpgradeOperationsToReturn = 2

type Handler struct {
	instancesDb         storage.Instances
	operationsDb        storage.Operations
	runtimeStatesDb     storage.RuntimeStates
	bindingsDb          storage.Bindings
	instancesArchivedDb storage.InstancesArchived
	converter           Converter
	defaultMaxPage      int
	provisionerClient   provisioner.Client
	k8sClient           client.Client
	kimConfig           broker.KimConfig
	logger              logrus.FieldLogger
}

func NewHandler(instanceDb storage.Instances, operationDb storage.Operations, runtimeStatesDb storage.RuntimeStates,
	instancesArchived storage.InstancesArchived, bindingsDb storage.Bindings, defaultMaxPage int, defaultRequestRegion string,
	provisionerClient provisioner.Client,
	k8sClient client.Client, kimConfig broker.KimConfig,
	logger logrus.FieldLogger) *Handler {
	return &Handler{
		instancesDb:         instanceDb,
		operationsDb:        operationDb,
		runtimeStatesDb:     runtimeStatesDb,
		bindingsDb:          bindingsDb,
		converter:           NewConverter(defaultRequestRegion),
		defaultMaxPage:      defaultMaxPage,
		provisionerClient:   provisionerClient,
		instancesArchivedDb: instancesArchived,
		kimConfig:           kimConfig,
		k8sClient:           k8sClient,
		logger:              logger.WithField("service", "RuntimeHandler"),
	}
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/runtimes", h.getRuntimes)
}

func unionInstances(sets ...[]pkg.RuntimeDTO) (union []pkg.RuntimeDTO) {
	m := make(map[string]pkg.RuntimeDTO)
	for _, s := range sets {
		for _, i := range s {
			if _, exists := m[i.InstanceID]; !exists {
				m[i.InstanceID] = i
			}
		}
	}
	for _, i := range m {
		union = append(union, i)
	}
	return
}

func (h *Handler) listInstances(filter dbmodel.InstanceFilter) ([]pkg.RuntimeDTO, int, int, error) {
	if slices.Contains(filter.States, dbmodel.InstanceDeprovisioned) {
		// try to list instances where deletion didn't finish successfully
		// entry in the Instances table still exists but has deletion timestamp and contains list of incomplete steps
		deletionAttempted := true
		filter.DeletionAttempted = &deletionAttempted
		instances, instancesCount, instancesTotalCount, _ := h.instancesDb.List(filter)

		instancesArchived, instancesArchivedCount, instancesArchivedTotalCount, err := h.instancesArchivedDb.List(filter)
		if err != nil {
			return []pkg.RuntimeDTO{}, instancesArchivedCount, instancesArchivedTotalCount, err
		}

		// return union of all sets of instances
		instanceDTOs := []pkg.RuntimeDTO{}
		for _, i := range instances {
			dto, err := h.converter.NewDTO(i)
			if err != nil {
				return []pkg.RuntimeDTO{}, instancesCount, instancesTotalCount, err
			}
			instanceDTOs = append(instanceDTOs, dto)
		}
		archived := []pkg.RuntimeDTO{}
		for _, i := range instancesArchived {
			instance := h.InstanceFromInstanceArchived(i)
			dto, err := h.converter.NewDTO(instance)
			if err != nil {
				return archived, instancesArchivedCount, instancesArchivedTotalCount, err
			}
			dto.Status = pkg.RuntimeStatus{
				CreatedAt: i.ProvisioningStartedAt,
				DeletedAt: &i.LastDeprovisioningFinishedAt,
				Provisioning: &pkg.Operation{
					CreatedAt: i.ProvisioningStartedAt,
					UpdatedAt: i.ProvisioningFinishedAt,
					State:     string(i.ProvisioningState),
				},
				Deprovisioning: &pkg.Operation{
					UpdatedAt: i.LastDeprovisioningFinishedAt,
				},
			}
			archived = append(archived, dto)
		}
		instancesUnion := unionInstances(instanceDTOs, archived)
		return instancesUnion, instancesCount + instancesArchivedCount, instancesTotalCount + instancesArchivedTotalCount, nil
	}

	var result []pkg.RuntimeDTO
	instances, count, total, err := h.instancesDb.List(filter)
	if err != nil {
		return []pkg.RuntimeDTO{}, 0, 0, err
	}
	for _, instance := range instances {
		dto, err := h.converter.NewDTO(instance)
		if err != nil {
			return []pkg.RuntimeDTO{}, 0, 0, err
		}
		result = append(result, dto)
	}
	return result, count, total, nil
}

func (h *Handler) InstanceFromInstanceArchived(archived internal.InstanceArchived) internal.Instance {
	return internal.Instance{
		InstanceID:                  archived.InstanceID,
		RuntimeID:                   archived.LastRuntimeID,
		GlobalAccountID:             archived.GlobalAccountID,
		SubscriptionGlobalAccountID: archived.SubscriptionGlobalAccountID,
		SubAccountID:                archived.SubaccountID,
		ServiceID:                   broker.KymaServiceID,
		ServiceName:                 broker.KymaServiceName,
		ServicePlanID:               archived.PlanID,
		ServicePlanName:             archived.PlanName,
		ProviderRegion:              archived.Region,
		CreatedAt:                   archived.ProvisioningStartedAt,
		Provider:                    internal.CloudProvider(archived.Provider),
		Reconcilable:                false,

		InstanceDetails: internal.InstanceDetails{
			ShootName: archived.ShootName,
		},

		Parameters: internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{
				UserID: archived.UserID(),
			},
			Parameters:     internal.ProvisioningParametersDTO{},
			PlatformRegion: archived.SubaccountRegion,
		},
	}
}

func (h *Handler) getRuntimes(w http.ResponseWriter, req *http.Request) {
	toReturn := make([]pkg.RuntimeDTO, 0)

	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(req, h.defaultMaxPage)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("unable to extract pagination: %s", err.Error()))
		httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("while getting query parameters: %w", err))
		return
	}
	filter := h.getFilters(req)
	filter.PageSize = pageSize
	filter.Page = page
	opDetail := getOpDetail(req)
	kymaConfig := getBoolParam(pkg.KymaConfigParam, req)
	clusterConfig := getBoolParam(pkg.ClusterConfigParam, req)
	gardenerConfig := getBoolParam(pkg.GardenerConfigParam, req)
	runtimeResourceConfig := getBoolParam(pkg.RuntimeConfigParam, req)
	bindings := getBoolParam(pkg.BindingsParam, req)

	instances, count, totalCount, err := h.listInstances(filter)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("unable to fetch instances: %s", err.Error()))
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while fetching instances: %s", err.Error()))
		return
	}

	for _, dto := range instances {

		switch opDetail {
		case pkg.AllOperation:
			err = h.setRuntimeAllOperations(&dto)
		case pkg.LastOperation:
			err = h.setRuntimeLastOperation(&dto)
		}
		if err != nil {
			h.logger.Warn(fmt.Sprintf("unable to set operations: %s", err.Error()))
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		err = h.determineStatusModifiedAt(&dto)
		if err != nil {
			h.logger.Warn(fmt.Sprintf("unable to determine status: %s", err.Error()))
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		instanceDrivenByKimOnly := h.kimConfig.IsDrivenByKimOnly(dto.ServicePlanName)

		err = h.setRuntimeOptionalAttributes(&dto, kymaConfig, clusterConfig, gardenerConfig, instanceDrivenByKimOnly)
		if err != nil {
			h.logger.Warn(fmt.Sprintf("unable to set optional attributes: %s", err.Error()))
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if runtimeResourceConfig && dto.RuntimeID != "" {
			runtimeResourceName, runtimeNamespaceName := h.getRuntimeNamesFromLastOperation(dto)

			runtimeResourceObject := &unstructured.Unstructured{}
			runtimeResourceObject.SetGroupVersionKind(RuntimeResourceGVK())
			err = h.k8sClient.Get(context.Background(), client.ObjectKey{
				Namespace: runtimeNamespaceName,
				Name:      runtimeResourceName,
			}, runtimeResourceObject)
			switch {
			case errors.IsNotFound(err):
				h.logger.Info(fmt.Sprintf("Runtime resource %s/%s: is not found: %s", dto.InstanceID, dto.RuntimeID, err.Error()))
				dto.RuntimeConfig = nil
			case err != nil:
				h.logger.Warn(fmt.Sprintf("unable to get Runtime resource %s/%s: %s", dto.InstanceID, dto.RuntimeID, err.Error()))
				dto.RuntimeConfig = nil
			default:
				// remove managedFields from the object to reduce the size of the response
				_, ok := runtimeResourceObject.Object["metadata"].(map[string]interface{})
				if !ok {
					h.logger.Warn(fmt.Sprintf("unable to get Runtime resource metadata %s/%s: %s", dto.InstanceID, dto.RuntimeID, err.Error()))
					dto.RuntimeConfig = nil

				} else {
					delete(runtimeResourceObject.Object["metadata"].(map[string]interface{}), "managedFields")
					dto.RuntimeConfig = &runtimeResourceObject.Object
				}
			}

		}
		if bindings {
			err := h.addBindings(&dto)
			if err != nil {
				h.logger.Warn(fmt.Sprintf("unable to apply bindings: %s", err.Error()))
				httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		toReturn = append(toReturn, dto)
	}

	runtimePage := pkg.RuntimesPage{
		Data:       toReturn,
		Count:      count,
		TotalCount: totalCount,
	}
	httputil.WriteResponse(w, http.StatusOK, runtimePage)
}

func (h *Handler) getRuntimeNamesFromLastOperation(dto pkg.RuntimeDTO) (string, string) {
	// TODO get rid of additional DB query - we have this info fetched from DB but it is tedious to pass it through
	op, err := h.operationsDb.GetLastOperation(dto.InstanceID)
	runtimeResourceName := steps.KymaRuntimeResourceNameFromID(dto.RuntimeID)
	runtimeNamespaceName := "kcp-system"
	if err != nil || op.RuntimeResourceName != "" {
		runtimeResourceName = op.RuntimeResourceName
	}
	if err != nil || op.KymaResourceNamespace != "" {
		runtimeNamespaceName = op.KymaResourceNamespace
	}
	return runtimeResourceName, runtimeNamespaceName
}

func (h *Handler) takeLastNonDryRunClusterOperations(oprs []internal.UpgradeClusterOperation) ([]internal.UpgradeClusterOperation, int) {
	toReturn := make([]internal.UpgradeClusterOperation, 0)
	totalCount := 0
	for _, op := range oprs {
		if op.DryRun {
			continue
		}
		if len(toReturn) < numberOfUpgradeOperationsToReturn {
			toReturn = append(toReturn, op)
		}
		totalCount = totalCount + 1
	}
	return toReturn, totalCount
}

func (h *Handler) determineStatusModifiedAt(dto *pkg.RuntimeDTO) error {
	// Determine runtime modifiedAt timestamp based on the last operation of the runtime
	last, err := h.operationsDb.GetLastOperation(dto.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching last operation for instance %s: %w", dto.InstanceID, err)
	}
	if last != nil {
		dto.Status.ModifiedAt = last.UpdatedAt
	}
	return nil
}

func (h *Handler) setRuntimeAllOperations(dto *pkg.RuntimeDTO) error {
	operationsGroup, err := h.operationsDb.ListOperationsByInstanceIDGroupByType(dto.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching operations for instance %s: %w", dto.InstanceID, err)
	}

	provOprs := operationsGroup.ProvisionOperations
	if len(provOprs) != 0 {
		firstProvOp := &provOprs[len(provOprs)-1]
		lastProvOp := provOprs[0]
		// Set AVS evaluation ID based on the data in the last provisioning operation
		dto.AVSInternalEvaluationID = lastProvOp.InstanceDetails.Avs.AvsEvaluationInternalId
		h.converter.ApplyProvisioningOperation(dto, firstProvOp)
		if len(provOprs) > 1 {
			h.converter.ApplyUnsuspensionOperations(dto, provOprs[:len(provOprs)-1])
		}
	}

	deprovOprs := operationsGroup.DeprovisionOperations
	var deprovOp *internal.DeprovisioningOperation
	if len(deprovOprs) != 0 {
		for _, op := range deprovOprs {
			if !op.Temporary {
				deprovOp = &op
				break
			}
		}
	}
	h.converter.ApplyDeprovisioningOperation(dto, deprovOp)
	h.converter.ApplySuspensionOperations(dto, deprovOprs)

	ucOprs := operationsGroup.UpgradeClusterOperations
	ucOprs, totalCount := h.takeLastNonDryRunClusterOperations(ucOprs)
	h.converter.ApplyUpgradingClusterOperations(dto, ucOprs, totalCount)

	uOprs := operationsGroup.UpdateOperations
	totalCount = len(uOprs)
	if len(uOprs) > numberOfUpgradeOperationsToReturn {
		uOprs = uOprs[0:numberOfUpgradeOperationsToReturn]
	}
	h.converter.ApplyUpdateOperations(dto, uOprs, totalCount)

	return nil
}

func (h *Handler) setRuntimeLastOperation(dto *pkg.RuntimeDTO) error {
	lastOp, err := h.operationsDb.GetLastOperation(dto.InstanceID)
	if err != nil {
		if dberr.IsNotFound(err) {
			h.logger.Infof("No operations found for instance %s", dto.InstanceID)
			return nil
		}
		return fmt.Errorf("while fetching last operation instance %s: %w", dto.InstanceID, err)
	}

	// Set AVS evaluation ID based on the data in the last operation
	dto.AVSInternalEvaluationID = lastOp.InstanceDetails.Avs.AvsEvaluationInternalId

	switch lastOp.Type {
	case internal.OperationTypeProvision:
		provOps, err := h.operationsDb.ListProvisioningOperationsByInstanceID(dto.InstanceID)
		if err != nil {
			return fmt.Errorf("while fetching provisioning operations for instance %s: %w", dto.InstanceID, err)
		}
		lastProvOp := &provOps[0]
		if len(provOps) > 1 {
			h.converter.ApplyUnsuspensionOperations(dto, []internal.ProvisioningOperation{*lastProvOp})
		} else {
			h.converter.ApplyProvisioningOperation(dto, lastProvOp)
		}

	case internal.OperationTypeDeprovision:
		deprovOp, err := h.operationsDb.GetDeprovisioningOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching deprovisioning operation for instance %s: %w", dto.InstanceID, err)
		}
		if deprovOp.Temporary {
			h.converter.ApplySuspensionOperations(dto, []internal.DeprovisioningOperation{*deprovOp})
		} else {
			h.converter.ApplyDeprovisioningOperation(dto, deprovOp)
		}

	case internal.OperationTypeUpgradeCluster:
		upgClusterOp, err := h.operationsDb.GetUpgradeClusterOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching upgrade cluster operation for instance %s: %w", dto.InstanceID, err)
		}
		h.converter.ApplyUpgradingClusterOperations(dto, []internal.UpgradeClusterOperation{*upgClusterOp}, 1)

	case internal.OperationTypeUpdate:
		updOp, err := h.operationsDb.GetUpdatingOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching update operation for instance %s: %w", dto.InstanceID, err)
		}
		h.converter.ApplyUpdateOperations(dto, []internal.UpdatingOperation{*updOp}, 1)

	default:
		return fmt.Errorf("unsupported operation type: %s", lastOp.Type)
	}

	return nil
}

func (h *Handler) setRuntimeOptionalAttributes(dto *pkg.RuntimeDTO, kymaConfig, clusterConfig, gardenerConfig, drivenByKimOnly bool) error {

	if kymaConfig || clusterConfig {
		states, err := h.runtimeStatesDb.ListByRuntimeID(dto.RuntimeID)
		if err != nil && !dberr.IsNotFound(err) {
			return fmt.Errorf("while fetching runtime states for instance %s: %w", dto.InstanceID, err)
		}
		for _, state := range states {
			if kymaConfig && dto.KymaConfig == nil && state.KymaConfig.Version != "" {
				config := state.KymaConfig
				dto.KymaConfig = &config
			}
			if clusterConfig && dto.ClusterConfig == nil && state.ClusterConfig.Provider != "" {
				config := state.ClusterConfig
				dto.ClusterConfig = &config
			}
			if dto.KymaConfig != nil && dto.ClusterConfig != nil {
				break
			}
		}
	}

	if gardenerConfig && dto.RuntimeID != "" && !drivenByKimOnly {
		runtimeStatus, err := h.provisionerClient.RuntimeStatus(dto.GlobalAccountID, dto.RuntimeID)
		if err != nil {
			dto.Status.GardenerConfig = nil
			h.logger.Warnf("unable to fetch runtime status for instance %s: %s", dto.InstanceID, err.Error())
		} else {
			dto.Status.GardenerConfig = runtimeStatus.RuntimeConfiguration.ClusterConfig
		}
	}

	return nil
}

func (h *Handler) getFilters(req *http.Request) dbmodel.InstanceFilter {
	var filter dbmodel.InstanceFilter
	query := req.URL.Query()
	// For optional filter, zero value (nil) is fine if not supplied
	filter.GlobalAccountIDs = query[pkg.GlobalAccountIDParam]
	filter.SubAccountIDs = query[pkg.SubAccountIDParam]
	filter.InstanceIDs = query[pkg.InstanceIDParam]
	filter.RuntimeIDs = query[pkg.RuntimeIDParam]
	filter.Regions = query[pkg.RegionParam]
	filter.Shoots = query[pkg.ShootParam]
	filter.Plans = query[pkg.PlanParam]
	if v, exists := query[pkg.ExpiredParam]; exists && v[0] == "true" {
		filter.Expired = ptr.Bool(true)
	}
	states := query[pkg.StateParam]
	if len(states) == 0 {
		// By default if no state filters are specified, suspended/deprovisioned runtimes are still excluded.
		filter.States = append(filter.States, dbmodel.InstanceNotDeprovisioned)
	} else {
		allState := false
		for _, s := range states {
			switch pkg.State(s) {
			case pkg.StateSucceeded:
				filter.States = append(filter.States, dbmodel.InstanceSucceeded)
			case pkg.StateFailed:
				filter.States = append(filter.States, dbmodel.InstanceFailed)
			case pkg.StateError:
				filter.States = append(filter.States, dbmodel.InstanceError)
			case pkg.StateProvisioning:
				filter.States = append(filter.States, dbmodel.InstanceProvisioning)
			case pkg.StateDeprovisioning:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioning)
			case pkg.StateUpgrading:
				filter.States = append(filter.States, dbmodel.InstanceUpgrading)
			case pkg.StateUpdating:
				filter.States = append(filter.States, dbmodel.InstanceUpdating)
			case pkg.StateSuspended:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioned)
			case pkg.StateDeprovisioned:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioned)
			case pkg.StateDeprovisionIncomplete:
				deletionAttempted := true
				filter.DeletionAttempted = &deletionAttempted
			case pkg.AllState:
				allState = true
			}
		}
		if allState {
			filter.States = nil
		}
	}

	return filter
}

func (h *Handler) addBindings(p *pkg.RuntimeDTO) error {
	bindings, err := h.bindingsDb.ListByInstanceID(p.InstanceID)
	if err != nil {
		return err
	}
	p.Bindings = make([]pkg.BindingDTO, 0, len(bindings))
	for _, b := range bindings {
		p.Bindings = append(p.Bindings, pkg.BindingDTO{
			ID:                b.ID,
			ExpirationSeconds: b.ExpirationSeconds,
			CreatedAt:         b.CreatedAt,
			ExpiresAt:         b.CreatedAt.Add(time.Duration(b.ExpirationSeconds) * time.Second),
			Type:              b.BindingType,
		})
	}

	return nil
}

func getOpDetail(req *http.Request) pkg.OperationDetail {
	opDetail := pkg.AllOperation
	opDetailParams := req.URL.Query()[pkg.OperationDetailParam]
	for _, p := range opDetailParams {
		opDetailParam := pkg.OperationDetail(p)
		switch opDetailParam {
		case pkg.AllOperation, pkg.LastOperation:
			opDetail = opDetailParam
		}
	}

	return opDetail
}

func getBoolParam(param string, req *http.Request) bool {
	requested := false
	params := req.URL.Query()[param]
	for _, p := range params {
		if p == "true" {
			requested = true
			break
		}
	}

	return requested
}

func RuntimeResourceGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "infrastructuremanager.kyma-project.io",
		Version: "v1",
		Kind:    "Runtime",
	}
}
