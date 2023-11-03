package expiration

import (
	"errors"
	"fmt"
	"net/http"
	"time"
	
	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	AttachRoutes(router *mux.Router)
}

type handler struct {
	instances  storage.Instances
	operations storage.Operations
	log        logrus.FieldLogger
}

func NewHandler(instancesStorage storage.Instances, operationsStorage storage.Operations, log logrus.FieldLogger) Handler {
	return &handler{
		instances:  instancesStorage,
		operations: operationsStorage,
		log:        log.WithField("service", "ExpirationEndpoint"),
	}
}

func (h *handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/expire/service_instance/{instance_id}", h.expireInstance).Methods("PUT")
}

func (h *handler) expireInstance(w http.ResponseWriter, req *http.Request) {
	instanceID := mux.Vars(req)["instance_id"]
	
	h.log.Info("Expiration triggered for instanceID: %s", instanceID)
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
	
	if instance.ServicePlanID != broker.TrialPlanID {
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
	
	instance, err = h.deactivateInstance(instance, logger)
	if err != nil {
		logger.Errorf("unable to update the instance in the database after deactivating: %s", err.Error())
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

func (h *handler) setInstanceExpirationTime(instance *internal.Instance, log logrus.FieldLogger) (*internal.Instance, error) {
	if instance.IsExpired() {
		log.Infof("instance expiration time has been already set at %s", instance.ExpiredAt.String())
		return instance, nil
	}
	log.Infof("setting expiration time for the instance created at %s", instance.CreatedAt)
	instance.ExpiredAt = ptr.Time(time.Now())
	instance, err := h.instances.Update(*instance)
	return instance, err
}

func (h *handler) deactivateInstance(instance *internal.Instance, log logrus.FieldLogger) (*internal.Instance, error) {
	active := instance.Parameters.ErsContext.Active
	if active != nil && !(*active) {
		log.Info("instance is already deactivated")
		return instance, nil
	}
	log.Info("deactivating the instance")
	*instance.Parameters.ErsContext.Active = false
	instance, err := h.instances.Update(*instance)
	return instance, err
}
