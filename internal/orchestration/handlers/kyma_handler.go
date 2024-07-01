package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
)

type kymaHandler struct{}

func NewKymaHandler() *kymaHandler {
	return &kymaHandler{}
}

func (h *kymaHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/upgrade/kyma", h.createOrchestration).Methods(http.MethodPost)
}

func (h *kymaHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("kyma upgrade not supported"))
}
