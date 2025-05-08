package additionalproperties

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
)

const ProvisioningRequestsFileName = "provisioning-requests.jsonl"

type Handler struct {
	logger                   *slog.Logger
	additionalPropertiesPath string
}

func NewHandler(logger *slog.Logger, additionalPropertiesPath string) *Handler {
	return &Handler{
		logger:                   logger.With("service", "additional-properties-handler"),
		additionalPropertiesPath: additionalPropertiesPath,
	}
}

func (h *Handler) AttachRoutes(router *httputil.Router) {
	router.HandleFunc("/additional_properties", h.getAdditionalProperties)
}

func (h *Handler) getAdditionalProperties(w http.ResponseWriter, req *http.Request) {
	filePath := filepath.Join(h.additionalPropertiesPath, ProvisioningRequestsFileName)

	f, err := os.Open(filePath)
	if err != nil {
		h.logger.Error("Failed to open additional properties file", "error", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while opening additional properties file: %s", err.Error()))
		return
	}
	defer f.Close()

	var records []map[string]interface{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			h.logger.Error("Failed to parse a line from additional properties", "error", err)
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		h.logger.Error("Error reading additional properties file", "error", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while reading additional properties file: %s", err.Error()))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, records)
}
