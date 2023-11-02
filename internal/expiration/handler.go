package expiration

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
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

	logger := h.log.WithField("instanceID", instanceID)
	logger.Info("Expiration triggered")
}
