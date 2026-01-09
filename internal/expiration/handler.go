package expiration

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/suspension"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type expirationResponse struct {
	SuspensionOpID string `json:"operation"`
}

type Handler interface {
	AttachRoutes(r router)
}

type handler struct {
	instances           storage.Instances
	operations          storage.Operations
	deprovisioningQueue suspension.Adder
	log                 *slog.Logger
}

func NewHandler(instancesStorage storage.Instances, operationsStorage storage.Operations, deprovisioningQueue suspension.Adder, log *slog.Logger) Handler {
	return &handler{
		instances:           instancesStorage,
		operations:          operationsStorage,
		deprovisioningQueue: deprovisioningQueue,
		log:                 log.With("service", "ExpirationEndpoint"),
	}
}

func (h *handler) AttachRoutes(r router) {
	r.HandleFunc("PUT /expire/service_instance/{instance_id}", h.expireInstance)
}

func (h *handler) expireInstance(w http.ResponseWriter, req *http.Request) {
	instanceID := req.PathValue("instance_id")

	h.log.Info(fmt.Sprintf("Expiration triggered for instanceID: %s", instanceID))
	logger := h.log.With("instanceID", instanceID)

	instance, err := h.instances.GetByID(instanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to get instance: %s", err.Error()))
		switch {
		case dberr.IsNotFound(err):
			httputil.WriteErrorResponse(w, http.StatusNotFound, err)
		default:
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}
	logger = logger.With("planName", instance.ServicePlanName)

	if instance.ServicePlanID != broker.TrialPlanID && instance.ServicePlanID != broker.FreemiumPlanID {
		msg := fmt.Sprintf("unsupported plan: %s", broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(instance.ServicePlanID)))
		logger.Warn(msg)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.New(msg))
		return
	}

	instance, err = h.setInstanceExpirationTime(instance, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to update the instance in the database after setting expiration time: %s", err.Error()))
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	instance, suspensionOpID, err := h.suspendInstance(instance, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to create suspension operation: %s", err.Error()))
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	instance, err = h.deactivateInstance(instance, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to update the instance in the database after deactivating: %s", err.Error()))
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	res := expirationResponse{suspensionOpID}
	httputil.WriteResponse(w, http.StatusAccepted, res)

	return
}

func (h *handler) setInstanceExpirationTime(instance *internal.Instance, log *slog.Logger) (*internal.Instance, error) {
	if instance.IsExpired() {
		log.Info(fmt.Sprintf("instance expiration time has been already set at %s", instance.ExpiredAt.String()))
		return instance, nil
	}
	log.Info(fmt.Sprintf("setting expiration time for the instance created at %s", instance.CreatedAt))
	instance.ExpiredAt = ptr.Time(time.Now().UTC())
	instance, err := h.instances.Update(*instance)
	return instance, err
}

func (h *handler) suspendInstance(instance *internal.Instance, log *slog.Logger) (*internal.Instance, string, error) {
	lastDeprovisioningOp, err := h.operations.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return instance, "", err
	}

	if lastDeprovisioningOp != nil {
		opType := "deprovisioning"
		if lastDeprovisioningOp.Temporary {
			opType = "suspension"
		}
		switch lastDeprovisioningOp.State {
		case internal.OperationStatePending:
			log.Info(fmt.Sprintf("%s pending", opType))
			return instance, lastDeprovisioningOp.ID, nil
		case domain.InProgress:
			log.Info(fmt.Sprintf("%s in progress", opType))
			return instance, lastDeprovisioningOp.ID, nil
		case domain.Failed:
			log.Info(fmt.Sprintf("triggering suspension after previous failed %s", opType))
		case domain.Succeeded:
			if len(lastDeprovisioningOp.ExcutedButNotCompleted) == 0 {
				log.Info("no steps to retry - not creating a new operation")
				return instance, lastDeprovisioningOp.ID, nil
			} else {
				log.Info(fmt.Sprintf("triggering suspension after previous %s with steps to retry", opType))
			}
		}
	}

	opID := uuid.New().String()
	suspensionOp := internal.NewSuspensionOperationWithID(opID, instance)
	if err := h.operations.InsertDeprovisioningOperation(suspensionOp); err != nil {
		return instance, "", err
	}
	h.deprovisioningQueue.Add(suspensionOp.ID)
	log.Info(fmt.Sprintf("suspension operation %s added to queue", suspensionOp.ID))

	return instance, suspensionOp.ID, nil
}

func (h *handler) deactivateInstance(instance *internal.Instance, log *slog.Logger) (*internal.Instance, error) {
	active := instance.Parameters.ErsContext.Active
	if active != nil && !(*active) {
		log.Info("instance is already deactivated")
		return instance, nil
	}
	log.Info("deactivating the instance")
	instance.Parameters.ErsContext.Active = ptr.Bool(false)
	instance, err := h.instances.Update(*instance)
	return instance, err
}
