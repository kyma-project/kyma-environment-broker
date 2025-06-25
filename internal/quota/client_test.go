package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQuota_Success(t *testing.T) {
	// given
	expectedQuota := 2
	expectedPlan := "aws"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Plan:  expectedPlan,
			Quota: expectedQuota,
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := newTestClient(server)

	// when
	quota, err := client.GetQuota("test-subaccount", expectedPlan)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expectedQuota, quota)
}

func TestGetQuota_WrongPlan(t *testing.T) {
	// given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Plan:  "different-plan",
			Quota: 100,
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := newTestClient(server)

	// when
	quota, err := client.GetQuota("test-subaccount", "expected-plan")

	// then
	assert.NoError(t, err)
	assert.Zero(t, quota)
}

func TestGetQuota_APIError(t *testing.T) {
	// given
	apiErrMsg := "not authorized"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		err := json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": apiErrMsg},
		})
		require.NoError(t, err)
	}))
	defer server.Close()

	client := newTestClient(server)

	// when
	quota, err := client.GetQuota("test-subaccount", "aws")

	// then
	assert.EqualError(t, err, fmt.Sprintf("API error: %s", apiErrMsg))
	assert.Zero(t, quota)
}

func newTestClient(server *httptest.Server) *Client {
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

	return client
}
