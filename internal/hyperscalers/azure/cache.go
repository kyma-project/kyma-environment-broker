package azure

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
)

const (
	refreshInterval    = 1 * time.Hour
	cacheRetries       = 3
	cacheRetryInterval = 2 * time.Second
)

// AzureCredentials holds the decoded Azure service principal credentials
// extracted from a Gardener secret.
type AzureCredentials struct {
	ClientID       string
	ClientSecret   string
	TenantID       string
	SubscriptionID string
}

// SecretFetcher fetches and decodes the current Azure credentials from Gardener.
// Called on every cache refresh to pick up rotated credentials.
type SecretFetcher func() (AzureCredentials, error)

// SKUsClientFactory creates a ResourceSKUsAPI (SKU = Stock-Keeping Unit) from credentials.
// Replaceable in tests to avoid real Azure API calls.
type SKUsClientFactory func(subscriptionID string, credential *azidentity.ClientSecretCredential) (ResourceSKUsAPI, error)

func defaultSKUsClientFactory(subscriptionID string, credential *azidentity.ClientSecretCredential) (ResourceSKUsAPI, error) {
	return armcompute.NewResourceSKUsClient(subscriptionID, credential, nil)
}

// AzureCache is a global cache of available zones per region and machine type.
// It is filled lazily in the background after KEB starts and refreshed every hour.
// Only regions and machine types configured in providerSpec are cached.
// The secret is re-fetched on every refresh to handle credential rotation.
type AzureCache struct {
	mu                sync.RWMutex
	data              map[string]map[string][]string // region → machineType → zones
	providerSpec      *configuration.ProviderSpec
	secretFetcher     SecretFetcher
	skusClientFactory SKUsClientFactory
}

// NewAzureCache creates a new AzureCache and starts a background goroutine that fills
// all configured regions and refreshes them every hour. It does not block startup.
func NewAzureCache(ctx context.Context, providerSpec *configuration.ProviderSpec, secretFetcher SecretFetcher) *AzureCache {
	c := &AzureCache{
		data:              make(map[string]map[string][]string),
		providerSpec:      providerSpec,
		secretFetcher:     secretFetcher,
		skusClientFactory: defaultSKUsClientFactory,
	}
	go c.run(ctx)
	return c
}

// ZonesFor returns available zones for the given region and machine type.
// Returns nil if the cache is not yet ready for the region.
func (c *AzureCache) ZonesFor(region, machineType string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	regionCache, ok := c.data[region]
	if !ok {
		return nil
	}
	return regionCache[machineType]
}

// Ready reports whether a fill attempt for the given region has completed successfully.
// Returns true even if no matching machine types were found (empty map is a valid cache entry).
// Returns false if the region was never attempted or all retries failed — callers fall back to a live API call.
func (c *AzureCache) Ready(region string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[region]
	return ok
}

func (c *AzureCache) run(ctx context.Context) {
	c.fillAll(ctx)

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.fillAll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *AzureCache) fillAll(ctx context.Context) {
	creds, err := c.secretFetcher()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to fetch Azure credentials for cache refresh: %s", err))
		return
	}
	for _, region := range c.providerSpec.Regions(pkg.Azure) {
		if err := c.fillRegionWithRetry(ctx, creds, region); err != nil {
			slog.Error(fmt.Sprintf("failed to fill Azure zone cache for region %s after %d retries: %s", region, cacheRetries, err))
		}
	}
}

func (c *AzureCache) fillRegionWithRetry(ctx context.Context, creds AzureCredentials, region string) error {
	credential, err := azidentity.NewClientSecretCredential(creds.TenantID, creds.ClientID, creds.ClientSecret, nil)
	if err != nil {
		return fmt.Errorf("while creating Azure credential: %w", err)
	}
	skusClient, err := c.skusClientFactory(creds.SubscriptionID, credential)
	if err != nil {
		return fmt.Errorf("while creating Azure ResourceSKUs client: %w", err)
	}

	var lastErr error
	for i := 0; i < cacheRetries; i++ {
		if err := c.fillRegion(ctx, skusClient, region); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-time.After(cacheRetryInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

func (c *AzureCache) fillRegion(ctx context.Context, skusClient ResourceSKUsAPI, region string) error {
	slog.Info(fmt.Sprintf("filling Azure zone cache for region %s", region))

	supportedMachineTypes := make(map[string]struct{})
	for _, mt := range c.providerSpec.MachineTypes(pkg.Azure) {
		supportedMachineTypes[mt] = struct{}{}
	}

	zones, err := fetchZonesBySKU(ctx, skusClient, region, supportedMachineTypes)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.data[region] = zones
	c.mu.Unlock()

	slog.Info(fmt.Sprintf("Azure zone cache filled for region %s (%d machine types)", region, len(zones)))
	return nil
}

func fetchZonesBySKU(ctx context.Context, skusClient ResourceSKUsAPI, region string, supportedMachineTypes map[string]struct{}) (map[string][]string, error) {
	result := make(map[string][]string)
	filter := fmt.Sprintf("location eq '%s'", region)
	pager := skusClient.NewListPager(&armcompute.ResourceSKUsClientListOptions{Filter: &filter})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Azure resource SKUs for region %s: %w", region, err)
		}
		for _, sku := range page.Value {
			if sku.ResourceType == nil || *sku.ResourceType != resourceTypeVirtualMachines || sku.Name == nil {
				continue
			}
			if _, ok := supportedMachineTypes[*sku.Name]; !ok {
				continue
			}
			result[*sku.Name] = availableZonesFromSKU(sku)
		}
	}
	return result, nil
}

// AzureCachedClient implements hyperscalers.ProviderClient using the global AzureCache.
// Used for HTTP validation and machines_availability — zero latency, no API calls.
type AzureCachedClient struct {
	cache        *AzureCache
	region       string
	providerSpec *configuration.ProviderSpec
}

func NewCachedClient(cache *AzureCache, region string, providerSpec *configuration.ProviderSpec) *AzureCachedClient {
	return &AzureCachedClient{cache: cache, region: region, providerSpec: providerSpec}
}

func (c *AzureCachedClient) AvailableZones(_ context.Context, machineType string) ([]string, error) {
	machineType = c.providerSpec.ResolveMachineType(pkg.Azure, machineType)
	return c.cache.ZonesFor(c.region, machineType), nil
}

func (c *AzureCachedClient) AvailableZonesCount(ctx context.Context, machineType string) (int, error) {
	zones, err := c.AvailableZones(ctx, machineType)
	return len(zones), err
}
