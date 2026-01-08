package version

import (
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
)

type Handler struct {
	version string
}

func NewHandler(version string) *Handler {
	return &Handler{
		version: version,
	}
}

func (h *Handler) AttachRoutes(router *httputil.Router) {
	router.HandleFunc("/version", h.getVersion)
}

func (h *Handler) getVersion(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteResponse(w, http.StatusOK, map[string]any{"version": h.version})
}
