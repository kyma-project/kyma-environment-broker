package azure

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// retries and interval are intentionally different from AWS (5×1s):
	// Azure ResourceSKUs API is slower and more prone to throttling,
	// so fewer retries with longer backoff are preferred.
	retries  = 3
	interval = 2 * time.Second

	resourceTypeVirtualMachines = "virtualMachines"
)

type ResourceSKUsAPI interface {
	NewListPager(options *armcompute.ResourceSKUsClientListOptions) *runtime.Pager[armcompute.ResourceSKUsClientListResponse]
}

type AzureClient struct {
	skusClient   ResourceSKUsAPI
	region       string
	providerSpec *configuration.ProviderSpec
	cache        map[string][]string
	cacheLoaded  bool
}

func NewClientFromSecret(ctx context.Context, providerSpec *configuration.ProviderSpec, secret *unstructured.Unstructured, region string) (*AzureClient, error) {
	creds, err := ExtractCredentials(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Azure credentials: %w", err)
	}

	credential, err := azidentity.NewClientSecretCredential(creds.TenantID, creds.ClientID, creds.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("while creating Azure credential: %w", err)
	}

	skusClient, err := armcompute.NewResourceSKUsClient(creds.SubscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("while creating Azure ResourceSKUs client: %w", err)
	}

	return &AzureClient{
		skusClient:   skusClient,
		region:       region,
		providerSpec: providerSpec,
	}, nil
}

func (c *AzureClient) AvailableZones(ctx context.Context, machineType string) ([]string, error) {
	machineType = c.providerSpec.ResolveMachineType(pkg.Azure, machineType)

	if err := c.ensureZonesLoaded(ctx); err != nil {
		return nil, err
	}

	return c.cache[machineType], nil
}

func (c *AzureClient) AvailableZonesCount(ctx context.Context, machineType string) (int, error) {
	zones, err := c.AvailableZones(ctx, machineType)
	if err != nil {
		return 0, err
	}
	return len(zones), nil
}

func (c *AzureClient) ensureZonesLoaded(ctx context.Context) error {
	if c.cacheLoaded {
		return nil
	}

	var err error
	for i := 0; i < retries; i++ {
		if err = c.tryFillCache(ctx); err == nil {
			return nil
		}
		c.cache = nil
		time.Sleep(interval)
	}
	return err
}

func (c *AzureClient) tryFillCache(ctx context.Context) error {
	supportedMachineTypes := make(map[string]struct{})
	for _, mt := range c.providerSpec.MachineTypes(pkg.Azure) {
		supportedMachineTypes[mt] = struct{}{}
	}
	slog.Info(fmt.Sprintf("querying Azure ResourceSKUs for region %s", c.region))
	zones, err := fetchZonesBySKU(ctx, c.skusClient, c.region, supportedMachineTypes)
	if err != nil {
		return err
	}
	c.cache = zones
	c.cacheLoaded = true
	slog.Info(fmt.Sprintf("Azure ResourceSKUs loaded for region %s (%d machine types cached)", c.region, len(c.cache)))
	return nil
}

// availableZonesFromSKU returns zones where the SKU is available,
// excluding zones covered by zone-type restrictions.
// Multiple restrictions of type Zone are all applied — each subtracts its zones from the available set.
func availableZonesFromSKU(sku *armcompute.ResourceSKU) []string {
	if len(sku.LocationInfo) == 0 {
		return nil
	}

	restricted := make(map[string]struct{})
	for _, restriction := range sku.Restrictions {
		if restriction.Type == nil || *restriction.Type != armcompute.ResourceSKURestrictionsTypeZone {
			continue
		}
		if restriction.RestrictionInfo == nil {
			continue
		}
		for _, z := range restriction.RestrictionInfo.Zones {
			if z != nil {
				restricted[*z] = struct{}{}
			}
		}
	}

	// LocationInfo[0] is the entry for the queried region — the pager is filtered
	// by location, so the API always returns exactly one LocationInfo element.
	var zones []string
	for _, z := range sku.LocationInfo[0].Zones {
		if z == nil {
			continue
		}
		if _, ok := restricted[*z]; !ok {
			zones = append(zones, *z)
		}
	}
	return zones
}

func ExtractSubscriptionID(secret *unstructured.Unstructured) (string, error) {
	data, found, err := unstructured.NestedStringMap(secret.Object, "data")
	if err != nil {
		return "", fmt.Errorf("unable to extract data from secret: %w", err)
	}
	if !found {
		return "", fmt.Errorf("secret does not contain data")
	}
	return extractField(data, "subscriptionID")
}

func ExtractCredentials(secret *unstructured.Unstructured) (AzureCredentials, error) {
	data, found, err := unstructured.NestedStringMap(secret.Object, "data")
	if err != nil {
		return AzureCredentials{}, fmt.Errorf("unable to extract data from secret: %w", err)
	}
	if !found {
		return AzureCredentials{}, fmt.Errorf("secret does not contain data")
	}

	clientID, err := extractField(data, "clientID")
	if err != nil {
		return AzureCredentials{}, err
	}
	clientSecret, err := extractField(data, "clientSecret")
	if err != nil {
		return AzureCredentials{}, err
	}
	tenantID, err := extractField(data, "tenantID")
	if err != nil {
		return AzureCredentials{}, err
	}
	subscriptionID, err := extractField(data, "subscriptionID")
	if err != nil {
		return AzureCredentials{}, err
	}

	return AzureCredentials{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	}, nil
}

func extractField(data map[string]string, field string) (string, error) {
	raw, ok := data[field]
	if !ok {
		return "", fmt.Errorf("secret does not contain %s", field)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("failed to decode %s: %w", field, err)
	}
	return string(decoded), nil
}
