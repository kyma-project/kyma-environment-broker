package hyperscalers

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/azure"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newFactoryTestCredentials() azure.AzureCredentials {
	return azure.AzureCredentials{
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
	}
}

func TestNewFactory_NoAzureCache(t *testing.T) {
	spec := newFactoryTestSpec(t)
	f := NewFactory(spec)
	factory := f.(*hyperscalerFactory)
	assert.Nil(t, factory.azureCache)
}

func TestNewFactoryWithAzureCache_NilFetcherNoCache(t *testing.T) {
	spec := newFactoryTestSpec(t)
	f := NewFactoryWithAzureCache(context.Background(), spec, nil)
	factory := f.(*hyperscalerFactory)
	assert.Nil(t, factory.azureCache, "azureCache must be nil when fetcher is nil")
}

func TestNewFactoryWithAzureCache_CacheCreatedWhenFetcherProvided(t *testing.T) {
	spec := newFactoryTestSpec(t)
	fetcher := func() (azure.AzureCredentials, error) { return newFactoryTestCredentials(), nil }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := NewFactoryWithAzureCache(ctx, spec, fetcher)
	factory := f.(*hyperscalerFactory)
	assert.NotNil(t, factory.azureCache)
}

func TestNewFromSecret_AzureFallsBackWhenCacheNil(t *testing.T) {
	spec := newFactoryTestSpec(t)
	f := &hyperscalerFactory{providerSpec: spec, azureCache: nil}

	client, err := f.NewFromSecret(context.Background(), pkg.Azure, newFactoryTestSecret(), "westeurope")
	require.NoError(t, err)
	_, isCached := client.(*azure.AzureCachedClient)
	assert.False(t, isCached, "must not return AzureCachedClient when azureCache is nil")
}

// TestNewFromSecret_AzureReturnsCachedClientWhenReady and
// TestNewPerCallFromSecret_AzureAlwaysPerCall require a pre-filled AzureCache.
// Since AzureCache.data is unexported, these tests live in azure/cache_test.go
// which has access to internal state via newTestCache.
// The dispatch logic itself (Ready → CachedClient, !Ready → AzureClient)
// is tested transitively through TestAzureCachedClient_ReturnsFromCache
// and TestAzureCachedClient_NotReadyRegionReturnsNil in cache_test.go.

func TestNewPerCallFromSecret_AzureNeverUsesCachedClient(t *testing.T) {
	// NewPerCallFromSecret must return per-call AzureClient even when azureCache is non-nil.
	// We use a non-nil cache (cache will never be ready since fill goroutine uses fake creds)
	// to prove cache presence alone doesn't trigger cached path.
	spec := newFactoryTestSpec(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := NewFactoryWithAzureCache(ctx, spec, func() (azure.AzureCredentials, error) {
		return newFactoryTestCredentials(), nil
	})

	client, err := f.NewPerCallFromSecret(context.Background(), pkg.Azure, newFactoryTestSecret(), "westeurope")
	require.NoError(t, err)
	_, isCached := client.(*azure.AzureCachedClient)
	assert.False(t, isCached, "NewPerCallFromSecret must never return AzureCachedClient")
}

func newFactoryTestSpec(t *testing.T) *configuration.ProviderSpec {
	t.Helper()
	spec, err := configuration.NewProviderSpec(strings.NewReader(`
azure:
  zonesDiscovery: true
  regions:
    westeurope:
      displayName: "West Europe"
  machines:
    Standard_D4s_v5: "Standard D4s v5"
`))
	require.NoError(t, err)
	return spec
}

func newFactoryTestSecret() *unstructured.Unstructured {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	s := &unstructured.Unstructured{}
	s.Object = map[string]interface{}{
		"data": map[string]interface{}{
			"clientID":       enc("test-client-id"),
			"clientSecret":   enc("test-client-secret"),
			"tenantID":       enc("test-tenant-id"),
			"subscriptionID": enc("test-subscription-id"),
		},
	}
	return s
}
