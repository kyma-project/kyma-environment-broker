package additionalproperties

import (
	"encoding/json"
	"fmt"
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

		var page Page
		err := json.Unmarshal(resp.Body.Bytes(), &page)
		require.NoError(t, err)
		require.Len(t, page.Data, 1)
		assert.Equal(t, 1, page.Count)
		assert.Equal(t, 1, page.TotalCount)

		var data map[string]interface{}
		err = json.Unmarshal([]byte(page.Data[0]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga1", data["globalAccountID"])
		assert.Equal(t, "sa1", data["subAccountID"])
		assert.Equal(t, "id1", data["instanceID"])
		payload := data["payload"].(map[string]interface{})
		assert.Equal(t, "provisioning1", payload["key"])
	})

	t.Run("returns update requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=update", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var page Page
		err := json.Unmarshal(resp.Body.Bytes(), &page)
		require.NoError(t, err)
		require.Len(t, page.Data, 1)
		assert.Equal(t, 1, page.Count)
		assert.Equal(t, 1, page.TotalCount)

		var data map[string]interface{}
		err = json.Unmarshal([]byte(page.Data[0]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga2", data["globalAccountID"])
		assert.Equal(t, "sa2", data["subAccountID"])
		assert.Equal(t, "id2", data["instanceID"])
		payload := data["payload"].(map[string]interface{})
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

func TestGetAdditionalProperties_Paging(t *testing.T) {
	tempDir := t.TempDir()

	provisioningFile := filepath.Join(tempDir, ProvisioningRequestsFileName)
	var content string
	for i := 1; i <= 5; i++ {
		line := fmt.Sprintf(`{"globalAccountID":"ga%d","subAccountID":"sa%d","instanceID":"id%d","payload":{"key":"p%d"}}`, i, i, i, i)
		content += line + "\n"
	}
	err := os.WriteFile(provisioningFile, []byte(content), 0644)
	require.NoError(t, err)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewHandler(log, tempDir)

	router := httputil.NewRouter()
	handler.AttachRoutes(router)

	t.Run("returns first page with pageSize=2", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=provisioning&pageSize=2&pageNumber=0", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var page Page
		err := json.Unmarshal(resp.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.Equal(t, 2, page.Count)
		assert.Equal(t, 5, page.TotalCount)

		var data map[string]interface{}
		err = json.Unmarshal([]byte(page.Data[0]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga1", data["globalAccountID"])

		err = json.Unmarshal([]byte(page.Data[1]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga2", data["globalAccountID"])
	})

	t.Run("returns second page with pageSize=2", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=provisioning&pageSize=2&pageNumber=1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var page Page
		err := json.Unmarshal(resp.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.Equal(t, 2, page.Count)
		assert.Equal(t, 5, page.TotalCount)

		var data map[string]interface{}
		err = json.Unmarshal([]byte(page.Data[0]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga3", data["globalAccountID"])

		err = json.Unmarshal([]byte(page.Data[1]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga4", data["globalAccountID"])
	})

	t.Run("returns last page with 1 item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=provisioning&pageSize=2&pageNumber=2", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var page Page
		err := json.Unmarshal(resp.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.Equal(t, 1, page.Count)
		assert.Equal(t, 5, page.TotalCount)

		var data map[string]interface{}
		err = json.Unmarshal([]byte(page.Data[0]), &data)
		require.NoError(t, err)
		assert.Equal(t, "ga5", data["globalAccountID"])
	})

	t.Run("returns empty data for out-of-range page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/additional_properties?requestType=provisioning&pageSize=2&pageNumber=3", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		var page Page
		err := json.Unmarshal(resp.Body.Bytes(), &page)
		require.NoError(t, err)
		assert.Equal(t, 0, page.Count)
		assert.Equal(t, 5, page.TotalCount)
		assert.Empty(t, page.Data)
	})
}
