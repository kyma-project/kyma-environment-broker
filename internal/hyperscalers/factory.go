package hyperscalers

import (
	"context"
	"fmt"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/azure"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type hyperscalerFactory struct {
	providerSpec *configuration.ProviderSpec
	azureCache   *azure.AzureCache
}

// NewFactory creates a new Factory without a global Azure zone cache.
// All zone discovery calls go directly to the hyperscaler API.
func NewFactory(providerSpec *configuration.ProviderSpec) Factory {
	return &hyperscalerFactory{providerSpec: providerSpec}
}

// NewFactoryWithAzureCache creates a Factory with a global Azure zone cache.
// The cache fills lazily in the background — KEB startup is not blocked.
// secretFetcher is called on every cache refresh to handle credential rotation.
// If secretFetcher is nil or Azure zones discovery is disabled, behaves like NewFactory.
func NewFactoryWithAzureCache(ctx context.Context, providerSpec *configuration.ProviderSpec, secretFetcher azure.SecretFetcher) Factory {
	var azureCache *azure.AzureCache
	if secretFetcher != nil && providerSpec.ZonesDiscovery(pkg.Azure) {
		azureCache = azure.NewAzureCache(ctx, providerSpec, secretFetcher)
	}
	return &hyperscalerFactory{
		providerSpec: providerSpec,
		azureCache:   azureCache,
	}
}

func (f *hyperscalerFactory) NewFromSecret(ctx context.Context, provider pkg.CloudProvider, secret *unstructured.Unstructured, region string) (ProviderClient, error) {
	switch provider {
	case pkg.AWS:
		return aws.NewClientFromSecret(ctx, f.providerSpec, secret, region)
	case pkg.Azure:
		// Use global cache if available and ready for this region — zero latency.
		// Falls back to per-call client if cache is not yet ready (lazy fill in progress).
		// Note: the cached client uses zone data from the startup secret, not the caller-provided
		// secret. This is intentional — the cache trades per-subscription accuracy for speed.
		if f.azureCache != nil && f.azureCache.Ready(region) {
			return azure.NewCachedClient(f.azureCache, region, f.providerSpec), nil
		}
		return azure.NewClientFromSecret(ctx, f.providerSpec, secret, region)
	default:
		return nil, fmt.Errorf("zone discovery not supported for provider %s", provider)
	}
}

// NewPerCallFromSecret always creates a fresh per-call client, bypassing the global cache.
// Used by the async DiscoverAvailableZonesCBStep to ensure zone discovery uses
// the exact Kyma-specific subscription secret for accurate per-subscription results.
func (f *hyperscalerFactory) NewPerCallFromSecret(ctx context.Context, provider pkg.CloudProvider, secret *unstructured.Unstructured, region string) (ProviderClient, error) {
	switch provider {
	case pkg.AWS:
		return aws.NewClientFromSecret(ctx, f.providerSpec, secret, region)
	case pkg.Azure:
		return azure.NewClientFromSecret(ctx, f.providerSpec, secret, region)
	default:
		return nil, fmt.Errorf("zone discovery not supported for provider %s", provider)
	}
}
