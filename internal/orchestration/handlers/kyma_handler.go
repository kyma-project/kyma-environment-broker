package handlers

import (
	"fmt"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
)

type kymaHandler struct{}

func NewKymaHandler() *kymaHandler {
	return &kymaHandler{}
}

func (h *kymaHandler) AttachRoutes(r router) {
	r.HandleFunc("POST /upgrade/kyma", h.createOrchestration)
}

func (h *kymaHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("kyma upgrade not supported"))
}
