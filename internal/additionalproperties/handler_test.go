package additionalproperties

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAdditionalProperties(t *testing.T) {
	tempDir := t.TempDir()

	provisioningFile := filepath.Join(tempDir, ProvisioningRequestsFileName)
	provisioningContent := `{"globalAccountID":"ga1","subAccountID":"sa1","instanceID":"id1","payload":{"key":"provisioning1"}}`
	err := os.WriteFile(provisioningFile, []byte(provisioningContent), 0644)
	require.NoError(t, err)

	updateFile := filepath.Join(tempDir, UpdateRequestsFileName)
	updateContent := `{"globalAccountID":"ga2","subAccountID":"sa2","instanceID":"id2","payload":{"key":"update1"}}`
	err = os.WriteFile(updateFile, []byte(updateContent), 0644)
	require.NoError(t, err)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewHandler(log, tempDir)

	router := httputil.NewRouter()
	handler.AttachRoutes(router)

	t.Run("returns provisioning requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=provisioning", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var data []map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.Len(t, data, 1)
		assert.Equal(t, "ga1", data[0]["globalAccountID"])
		assert.Equal(t, "sa1", data[0]["subAccountID"])
		assert.Equal(t, "id1", data[0]["instanceID"])
		payload := data[0]["payload"].(map[string]interface{})
		assert.Equal(t, "provisioning1", payload["key"])
	})

	t.Run("returns update requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=update", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var data []map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.Len(t, data, 1)
		assert.Equal(t, "ga2", data[0]["globalAccountID"])
		assert.Equal(t, "sa2", data[0]["subAccountID"])
		assert.Equal(t, "id2", data[0]["instanceID"])
		payload := data[0]["payload"].(map[string]interface{})
		assert.Equal(t, "update1", payload["key"])
	})

	t.Run("returns error for missing requestType", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusBadRequest, resp.Code)

		var data map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.Contains(t, data["message"], "Missing query parameter")
	})

	t.Run("returns error for invalid requestType", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=invalid", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusBadRequest, resp.Code)

		var data map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.Contains(t, data["message"], "Unsupported requestType")
	})
}
