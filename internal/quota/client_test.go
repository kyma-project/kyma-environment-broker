package quota

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQuota_Success(t *testing.T) {
	// given
	response := Response{
		Plan:  "aws",
		Quota: 2,
	}
	client, cleanup := fixClient(t, http.StatusOK, response)
	defer cleanup()

	// when
	quota, err := client.GetQuota("test-subaccount", response.Plan)

	// then
	assert.NoError(t, err)
	assert.Equal(t, response.Quota, quota)
}

func TestGetQuota_WrongPlan(t *testing.T) {
	// given
	response := Response{
		Plan:  "different-plan",
		Quota: 100,
	}
	client, cleanup := fixClient(t, http.StatusOK, response)
	defer cleanup()

	// when
	quota, err := client.GetQuota("test-subaccount", "expected-plan")

	// then
	assert.NoError(t, err)
	assert.Zero(t, quota)
}

func TestGetQuota_APIError(t *testing.T) {
	// given
	APIErr := map[string]any{
		"error": map[string]string{"message": "not authorized"},
	}
	client, cleanup := fixClient(t, http.StatusForbidden, APIErr)
	defer cleanup()

	// when
	quota, err := client.GetQuota("test-subaccount", "aws")

	// then
	assert.EqualError(t, err, "API error: not authorized")
	assert.Zero(t, quota)
}

func fixClient(t *testing.T, statusCode int, response any) (*Client, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	cfg := Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		AuthURL:      server.URL,
		ServiceURL:   server.URL,
	}
	client := &Client{
		ctx:        context.Background(),
		httpClient: server.Client(),
		config:     cfg,
		log:        slog.Default(),
	}
	cleanup := func() {
		server.Close()
	}
	return client, cleanup
}
