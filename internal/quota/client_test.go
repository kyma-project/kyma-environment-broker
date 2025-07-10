package quota

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestGetQuota_SubaccountNotExits(t *testing.T) {
	// given
	APIErr := map[string]any{
		"error": map[string]string{"message": "Tenant doesn't exist under subaccount [test-subaccount]"},
	}
	client, cleanup := fixClient(t, http.StatusNotFound, APIErr)
	defer cleanup()

	// when
	quota, err := client.GetQuota("test-subaccount", "aws")

	// then
	assert.EqualError(t, err, "Subaccount test-subaccount does not exist")
	assert.Zero(t, quota)
}

func TestGetQuota_ProvisioningServiceNotAvailable(t *testing.T) {
	// given
	client, cleanup := fixClient(t, http.StatusInternalServerError, map[string]any{})
	defer cleanup()

	// when
	quota, err := client.GetQuota("test-subaccount", "aws")

	// then
	assert.EqualError(t, err, "The provisioning service is currently unavailable. Please try again later")
	assert.Zero(t, quota)
}

func TestGetQuota_SuccessAfterProvisioningServiceRetry(t *testing.T) {
	// given
	callCount := 0
	response := Response{
		Plan:  "aws",
		Quota: 2,
	}

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
		require.NoError(t, err)
	}))
	defer authServer.Close()
	serviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			err := json.NewEncoder(w).Encode(map[string]any{})
			require.NoError(t, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer serviceServer.Close()

	client := NewClient(context.Background(), fixConfig(authServer.URL, serviceServer.URL), slog.Default())

	// when
	quota, err := client.GetQuota("test-subaccount", response.Plan)

	// then
	assert.NoError(t, err)
	assert.Equal(t, response.Quota, quota)
}

func TestGetQuota_AuthenticationServiceNotAvailable(t *testing.T) {
	// given
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		err := json.NewEncoder(w).Encode(map[string]interface{}{})
		require.NoError(t, err)
	}))
	defer authServer.Close()
	serviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serviceServer.Close()

	client := NewClient(context.Background(), fixConfig(authServer.URL, serviceServer.URL), slog.Default())

	// when
	quota, err := client.GetQuota("test-subaccount", "aws")

	// then
	assert.EqualError(t, err, "The authentication service is currently unavailable. Please try again later")
	assert.Zero(t, quota)
}

func TestGetQuota_SuccessAfterAuthenticationServiceRetry(t *testing.T) {
	// given
	callCount := 0
	response := Response{
		Plan:  "aws",
		Quota: 2,
	}

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			err := json.NewEncoder(w).Encode(map[string]any{})
			require.NoError(t, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
		require.NoError(t, err)
	}))
	defer authServer.Close()
	serviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	defer serviceServer.Close()

	client := NewClient(context.Background(), fixConfig(authServer.URL, serviceServer.URL), slog.Default())

	// when
	quota, err := client.GetQuota("test-subaccount", response.Plan)

	// then
	assert.NoError(t, err)
	assert.Equal(t, response.Quota, quota)
}

func fixClient(t *testing.T, statusCode int, response any) (*Client, func()) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
		require.NoError(t, err)
	}))
	serviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))
	client := NewClient(context.Background(), fixConfig(authServer.URL, serviceServer.URL), slog.Default())
	cleanup := func() {
		authServer.Close()
		serviceServer.Close()
	}
	return client, cleanup
}

func fixConfig(authURL, serviceURL string) Config {
	return Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		AuthURL:      authURL,
		ServiceURL:   serviceURL,
		Retries:      5,
		Interval:     10 * time.Millisecond,
	}
}
