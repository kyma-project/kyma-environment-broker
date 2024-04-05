package expiration

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/suspension"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type expirationResponse struct {
	SuspensionOpID string `json:"operation"`
}

type Handler interface {
	AttachRoutes(router *mux.Router)
}

type handler struct {
	instances           storage.Instances
	operations          storage.Operations
	deprovisioningQueue suspension.Adder
	log                 logrus.FieldLogger
}

func NewHandler(instancesStorage storage.Instances, operationsStorage storage.Operations, deprovisioningQueue suspension.Adder, log logrus.FieldLogger) Handler {
	return &handler{
		instances:           instancesStorage,
		operations:          operationsStorage,
		deprovisioningQueue: deprovisioningQueue,
		log:                 log.WithField("service", "ExpirationEndpoint"),
	}
}

func (h *handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/expire/service_instance/{instance_id}", h.expireInstance).Methods("PUT")
}

func (h *handler) expireInstance(w http.ResponseWriter, req *http.Request) {
	instanceID := mux.Vars(req)["instance_id"]

	h.log.Info("Expiration triggered for instanceID: ", instanceID)
	logger := h.log.WithField("instanceID", instanceID)

	instance, err := h.instances.GetByID(instanceID)
	if err != nil {
		logger.Errorf("unable to get instance: %s", err.Error())
		switch {
		case dberr.IsNotFound(err):
			httputil.WriteErrorResponse(w, http.StatusNotFound, err)
		default:
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}
	logger = logger.WithField("planName", instance.ServicePlanName)

	if instance.ServicePlanID != broker.TrialPlanID && instance.ServicePlanID != broker.FreemiumPlanID {
		msg := fmt.Sprintf("unsupported plan: %s", broker.PlanNamesMapping[instance.ServicePlanID])
		logger.Warn(msg)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.New(msg))
		return
	}

	instance, err = h.setInstanceExpirationTime(instance, logger)
	if err != nil {
		logger.Errorf("unable to update the instance in the database after setting expiration time: %s", err.Error())
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	instance, suspensionOpID, err := h.suspendInstance(instance, logger)
	if err != nil {
		logger.Errorf("unable to create suspension operation: %s", err.Error())
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	instance, err = h.deactivateInstance(instance, logger)
	if err != nil {
		logger.Errorf("unable to update the instance in the database after deactivating: %s", err.Error())
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	res := expirationResponse{suspensionOpID}
	httputil.WriteResponse(w, http.StatusAccepted, res)

	return
}

func (h *handler) setInstanceExpirationTime(instance *internal.Instance, log logrus.FieldLogger) (*internal.Instance, error) {
	if instance.IsExpired() {
		log.Infof("instance expiration time has been already set at %s", instance.ExpiredAt.String())
		return instance, nil
	}
	log.Infof("setting expiration time for the instance created at %s", instance.CreatedAt)
	instance.ExpiredAt = ptr.Time(time.Now().UTC())
	instance, err := h.instances.Update(*instance)
	return instance, err
}

func (h *handler) suspendInstance(instance *internal.Instance, log *logrus.Entry) (*internal.Instance, string, error) {
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
		case orchestration.Pending:
			log.Infof("%s pending", opType)
			return instance, lastDeprovisioningOp.ID, nil
		case domain.InProgress:
			log.Infof("%s in progress", opType)
			return instance, lastDeprovisioningOp.ID, nil
		case domain.Failed:
			log.Infof("triggering suspension after previous failed %s", opType)
		case domain.Succeeded:
			if len(lastDeprovisioningOp.ExcutedButNotCompleted) == 0 {
				log.Info("no steps to retry - not creating a new operation")
				return instance, lastDeprovisioningOp.ID, nil
			} else {
				log.Infof("triggering suspension after previous %s with steps to retry", opType)
			}
		}
	}

	opID := uuid.New().String()
	suspensionOp := internal.NewSuspensionOperationWithID(opID, instance)
	if err := h.operations.InsertDeprovisioningOperation(suspensionOp); err != nil {
		return instance, "", err
	}
	h.deprovisioningQueue.Add(suspensionOp.ID)
	log.Infof("suspension operation %s added to queue", suspensionOp.ID)

	return instance, suspensionOp.ID, nil
}

func (h *handler) deactivateInstance(instance *internal.Instance, log logrus.FieldLogger) (*internal.Instance, error) {
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
