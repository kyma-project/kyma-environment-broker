package additionalproperties

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
)

const (
	ProvisioningRequestsFileName = "provisioning-requests.jsonl"
	UpdateRequestsFileName       = "update-requests.jsonl"
)

type Handler struct {
	logger                   *slog.Logger
	additionalPropertiesPath string
}

type Page struct {
	Data       []string `json:"data"`
	Count      int      `json:"count"`
	TotalCount int      `json:"totalCount"`
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
	requestType := req.URL.Query().Get("requestType")
	pageNumberStr := req.URL.Query().Get("pageNumber")
	pageSizeStr := req.URL.Query().Get("pageSize")

	pageNumber := 0
	pageSize := 100

	if pageNumberStr != "" {
		if n, err := strconv.Atoi(pageNumberStr); err == nil && n >= 0 {
			pageNumber = n
		}
	}
	if pageSizeStr != "" {
		if s, err := strconv.Atoi(pageSizeStr); err == nil && s > 0 {
			pageSize = s
		}
	}

	var fileName string
	switch requestType {
	case "provisioning":
		fileName = ProvisioningRequestsFileName
	case "update":
		fileName = UpdateRequestsFileName
	case "":
		info := map[string]string{
			"message": "Missing query parameter 'requestType'. Supported values are 'provisioning' and 'update'.",
		}
		httputil.WriteResponse(w, http.StatusBadRequest, info)
		return
	default:
		info := map[string]string{
			"message": fmt.Sprintf("Unsupported requestType '%s'. Supported values are 'provisioning' and 'update'.", requestType),
		}
		httputil.WriteResponse(w, http.StatusBadRequest, info)
		return
	}
	filePath := filepath.Join(h.additionalPropertiesPath, fileName)

	f, err := os.Open(filePath)
	if err != nil {
		h.logger.Error("Failed to open additional properties file", "error", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while opening additional properties file: %s", err.Error()))
		return
	}
	defer func() { _ = f.Close() }()

	skip := pageNumber * pageSize
	end := skip + pageSize
	lineNumber := 0
	var pageData []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if lineNumber >= skip && lineNumber < end {
			pageData = append(pageData, string(scanner.Bytes()))
		}
		lineNumber++
	}
	if err := scanner.Err(); err != nil {
		h.logger.Error("Error reading additional properties file", "error", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while reading additional properties file: %s", err.Error()))
		return
	}

	page := Page{
		Data:       pageData,
		Count:      len(pageData),
		TotalCount: lineNumber,
	}

	httputil.WriteResponse(w, http.StatusOK, page)
}
