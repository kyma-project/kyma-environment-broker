package azure

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildAzureCredentials() AzureCredentials {
	return AzureCredentials{
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
	}
}

func buildCacheSpec(machineNames []string) *configuration.ProviderSpec {
	machines := ""
	for _, name := range machineNames {
		machines += "    \"" + name + "\": \"" + name + "\"\n"
	}
	yaml := "azure:\n  zonesDiscovery: true\n  regions:\n    westeurope:\n      displayName: \"West Europe\"\n  machines:\n" + machines
	spec, err := configuration.NewProviderSpec(strings.NewReader(yaml))
	if err != nil {
		panic(fmt.Sprintf("buildCacheSpec: failed to parse spec: %v", err))
	}
	return spec
}

// mockSKUsClientFactory returns a SKUsClientFactory that always returns the given API.
func mockSKUsClientFactory(api ResourceSKUsAPI) SKUsClientFactory {
	return func(_ string, _ *azidentity.ClientSecretCredential) (ResourceSKUsAPI, error) {
		return api, nil
	}
}

func newTestCache(spec *configuration.ProviderSpec, skus []*armcompute.ResourceSKU, apiErr error) *AzureCache {
	return &AzureCache{
		data:              make(map[string]map[string][]string),
		providerSpec:      spec,
		secretFetcher:     func() (AzureCredentials, error) { return buildAzureCredentials(), nil },
		skusClientFactory: mockSKUsClientFactory(&mockSKUsAPI{skus: skus, err: apiErr}),
	}
}

func newTestCacheWithMock(spec *configuration.ProviderSpec, api ResourceSKUsAPI) *AzureCache {
	return &AzureCache{
		data:              make(map[string]map[string][]string),
		providerSpec:      spec,
		skusClientFactory: mockSKUsClientFactory(api),
	}
}

func TestAzureCache_FillAndRead(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
		buildSKU("Standard_F8s_v2", []string{"1", "2"}, nil),
	}
	spec := buildCacheSpec([]string{"Standard_D4s_v5", "Standard_F8s_v2"})
	cache := newTestCacheWithMock(spec, &mockSKUsAPI{skus: skus})

	err := cache.fillRegion(context.Background(), &mockSKUsAPI{skus: skus}, "westeurope")
	require.NoError(t, err)

	assert.True(t, cache.Ready("westeurope"))
	assert.ElementsMatch(t, []string{"1", "2", "3"}, cache.ZonesFor("westeurope", "Standard_D4s_v5"))
	assert.ElementsMatch(t, []string{"1", "2"}, cache.ZonesFor("westeurope", "Standard_F8s_v2"))
	assert.Nil(t, cache.ZonesFor("westeurope", "Standard_Unknown"), "unknown machine type returns nil")
	assert.Nil(t, cache.ZonesFor("eastus", "Standard_D4s_v5"), "unfilled region returns nil")
}

func TestAzureCache_SecretFetcherError(t *testing.T) {
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})
	cache := &AzureCache{
		data:         make(map[string]map[string][]string),
		providerSpec: spec,
		secretFetcher: func() (AzureCredentials, error) {
			return AzureCredentials{}, assert.AnError
		},
		skusClientFactory: defaultSKUsClientFactory,
	}

	cache.fillAll(context.Background())

	assert.False(t, cache.Ready("westeurope"))
	assert.Nil(t, cache.ZonesFor("westeurope", "Standard_D4s_v5"))
}

func TestAzureCache_ContextCancellation(t *testing.T) {
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})
	skus := []*armcompute.ResourceSKU{buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil)}
	cache := newTestCache(spec, skus, nil)

	ctx, cancel := context.WithCancel(context.Background())
	goroutineDone := make(chan struct{})

	go func() {
		cache.run(ctx)
		close(goroutineDone)
	}()

	cancel()

	select {
	case <-goroutineDone:
		// goroutine stopped cleanly
	case <-time.After(3 * time.Second):
		t.Fatal("goroutine did not stop after context cancellation")
	}
}

func TestAzureCache_ConcurrentReads(t *testing.T) {
	// Run with: go test -race ./internal/hyperscalers/azure/...
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
	}
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})
	api := &mockSKUsAPI{skus: skus}
	cache := newTestCacheWithMock(spec, api)

	require.NoError(t, cache.fillRegion(context.Background(), api, "westeurope"))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			zones := cache.ZonesFor("westeurope", "Standard_D4s_v5")
			assert.NotNil(t, zones)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = cache.fillRegion(context.Background(), api, "westeurope")
	}()
	wg.Wait()
}

func TestAzureCachedClient_ReturnsFromCache(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
	}
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})
	api := &mockSKUsAPI{skus: skus}
	cache := newTestCacheWithMock(spec, api)

	require.NoError(t, cache.fillRegion(context.Background(), api, "westeurope"))

	client := NewCachedClient(cache, "westeurope", spec)

	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, zones)

	count, err := client.AvailableZonesCount(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestAzureCachedClient_NotReadyRegionReturnsNil(t *testing.T) {
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})
	cache := &AzureCache{
		data:         make(map[string]map[string][]string),
		providerSpec: spec,
	}

	client := NewCachedClient(cache, "westeurope", spec)
	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.Nil(t, zones, "nil zones when cache not ready — caller uses fallback per-call client")
}

func TestAzureCache_FillAllRetryLogsOnlyAfterAllAttempts(t *testing.T) {
	// fillAll should not mark a region as failed if a later retry succeeds.
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})

	var attempts int
	failThenSucceedAPI := &callbackMockSKUsAPI{
		callback: func() ([]*armcompute.ResourceSKU, error) {
			attempts++
			if attempts == 1 {
				return nil, assert.AnError
			}
			return []*armcompute.ResourceSKU{
				buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
			}, nil
		},
	}

	cache := &AzureCache{
		data:              make(map[string]map[string][]string),
		providerSpec:      spec,
		secretFetcher:     func() (AzureCredentials, error) { return buildAzureCredentials(), nil },
		skusClientFactory: mockSKUsClientFactory(failThenSucceedAPI),
	}

	cache.fillAll(context.Background())

	assert.True(t, cache.Ready("westeurope"), "region should be ready after successful retry")
}

func TestAzureCache_FillAllRetryExhausted(t *testing.T) {
	// fillAll logs error only after all retries are exhausted.
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})

	cache := &AzureCache{
		data:         make(map[string]map[string][]string),
		providerSpec: spec,
		secretFetcher: func() (AzureCredentials, error) {
			return buildAzureCredentials(), nil
		},
		skusClientFactory: func(_ string, _ *azidentity.ClientSecretCredential) (ResourceSKUsAPI, error) {
			return &mockSKUsAPI{err: assert.AnError}, nil
		},
	}

	cache.fillAll(context.Background())

	assert.False(t, cache.Ready("westeurope"), "region must not be ready when all retries fail")
}

func TestAzureCache_ClientFactoryError_NoRetry(t *testing.T) {
	spec := buildCacheSpec([]string{"Standard_D4s_v5"})
	var attempts int
	cache := &AzureCache{
		data:          make(map[string]map[string][]string),
		providerSpec:  spec,
		secretFetcher: func() (AzureCredentials, error) { return buildAzureCredentials(), nil },
		skusClientFactory: func(_ string, _ *azidentity.ClientSecretCredential) (ResourceSKUsAPI, error) {
			attempts++
			return nil, assert.AnError
		},
	}

	cache.fillAll(context.Background())

	assert.Equal(t, 1, attempts, "skusClientFactory must not be retried on client creation error")
	assert.False(t, cache.Ready("westeurope"))
}
