package kubeconfig

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kennygrant/sanitize"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

const attachmentName = "kubeconfig.yaml"

//go:generate mockery --name=KcBuilder --output=automock --outpkg=automock --case=underscore

type KcBuilder interface {
	Build(*internal.Instance) (string, error)
	GetServerURL(runtimeID string) (string, error)
}

type router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

type Handler struct {
	kubeconfigBuilder KcBuilder
	allowOrigins      string
	instanceStorage   storage.Instances
	operationStorage  storage.Operations
	log               *slog.Logger
}

func NewHandler(storage storage.BrokerStorage, b KcBuilder, origins string, log *slog.Logger) *Handler {
	return &Handler{
		instanceStorage:   storage.Instances(),
		operationStorage:  storage.Operations(),
		kubeconfigBuilder: b,
		allowOrigins:      origins,
		log:               log,
	}
}

func (h *Handler) AttachRoutes(r router) {
	r.HandleFunc("GET /kubeconfig/{instance_id}", h.GetKubeconfig)
	r.HandleFunc("GET /kubeconfig/", h.GetKubeconfig)
}

type ErrorResponse struct {
	Error string
}

func (h *Handler) GetKubeconfig(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("instance_id")
	if instanceID == "" {
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("instanceID is required"))
		return
	}

	h.specifyAllowOriginHeader(r, w)

	instance, err := h.instanceStorage.GetByID(instanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("instance with ID %s does not exist", instanceID))
		return
	default:
		h.log.Error(fmt.Sprintf("while getting instance for a kubeconfig, error: %s", err))
		h.handleResponse(w, http.StatusInternalServerError, err)
		return
	}

	if instance.RuntimeID == "" {
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("kubeconfig for instance %s does not exist. Provisioning could be in progress, please try again later", instanceID))
		return
	}

	operation, err := h.operationStorage.GetProvisioningOperationByInstanceID(instanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("provisioning operation for instance with ID %s does not exist", instanceID))
		return
	default:
		h.log.Error(fmt.Sprintf("while getting provision operation for kubeconfig, error: %s", err))
		h.handleResponse(w, http.StatusInternalServerError, err)
		return
	}

	if operation.InstanceID != instanceID {
		h.handleResponse(w, http.StatusBadRequest, fmt.Errorf("mismatch between operation and instance"))
		return
	}

	switch operation.State {
	case domain.InProgress, internal.OperationStatePending:
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("provisioning operation for instance %s is in progress state, kubeconfig not exist yet, please try again later", instanceID))
		return
	case domain.Failed:
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("provisioning operation for instance %s failed, kubeconfig does not exist", instanceID))
		return
	}

	newKubeconfig, err := h.kubeconfigBuilder.Build(instance)
	if err != nil {
		msgFmt := "while building kubeconfig: %s"
		if IsNotFound(err) {
			h.log.Info(fmt.Sprintf(msgFmt, err))
			h.handleResponse(w, http.StatusNotFound, errors.New("kubeconfig does not exist"))
			return
		}
		h.log.Error(fmt.Sprintf(msgFmt, err))
		h.handleResponse(w, http.StatusInternalServerError, fmt.Errorf("cannot fetch SKR kubeconfig: %s", err))
		return
	}

	writeToResponse(w, newKubeconfig, h.log)
}

func (h *Handler) handleResponse(w http.ResponseWriter, code int, err error) {
	errEncode := httputil.JSONEncodeWithCode(w, &ErrorResponse{Error: err.Error()}, code)
	if errEncode != nil {
		h.log.Error(fmt.Sprintf("cannot encode error response: %s", errEncode))
	}
}

func (h *Handler) specifyAllowOriginHeader(r *http.Request, w http.ResponseWriter) {
	origin := r.Header.Get("Origin")
	origin = strings.ReplaceAll(origin, "\r", "")
	origin = strings.ReplaceAll(origin, "\n", "")
	if origin == "" {
		return
	}

	if h.allowOrigins == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}

	for _, o := range strings.Split(h.allowOrigins, ",") {
		if o == origin {
			w.Header().Set("Access-Control-Allow-Origin", sanitize.HTML(origin))
			return
		}
	}
}

func writeToResponse(w http.ResponseWriter, data string, l *slog.Logger) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", attachmentName))
	w.Header().Add("Content-Type", "application/x-yaml")
	_, err := w.Write([]byte(data))
	if err != nil {
		l.Error(fmt.Sprintf("cannot write response with new kubeconfig: %s", err))
	}
}
