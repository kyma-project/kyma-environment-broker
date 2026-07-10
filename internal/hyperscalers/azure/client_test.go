package azure

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestExtractCredentials(t *testing.T) {
	secret := buildSecret(map[string]string{
		"clientID":       "client-id",
		"clientSecret":   "client-secret",
		"tenantID":       "tenant-id",
		"subscriptionID": "subscription-id",
	})

	creds, err := ExtractCredentials(secret)

	require.NoError(t, err)
	assert.Equal(t, "client-id", creds.ClientID)
	assert.Equal(t, "client-secret", creds.ClientSecret)
	assert.Equal(t, "tenant-id", creds.TenantID)
	assert.Equal(t, "subscription-id", creds.SubscriptionID)
}

func TestExtractCredentials_MissingField(t *testing.T) {
	secret := buildSecret(map[string]string{
		"clientID": "client-id",
		// missing other fields
	})

	_, err := ExtractCredentials(secret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientSecret")
}

func TestExtractSubscriptionID_Success(t *testing.T) {
	secret := buildSecret(map[string]string{"subscriptionID": "my-sub-id"})
	id, err := ExtractSubscriptionID(secret)
	require.NoError(t, err)
	assert.Equal(t, "my-sub-id", id)
}

func TestExtractSubscriptionID_NoData(t *testing.T) {
	// Secret with no data field — must return explicit error, not a nil-wrapped one.
	s := &unstructured.Unstructured{}
	s.Object = map[string]interface{}{}
	_, err := ExtractSubscriptionID(s)
	require.Error(t, err)
	assert.Equal(t, "secret does not contain data", err.Error())
}

func TestAvailableZones_HappyPath(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
		buildSKU("Standard_D8s_v5", []string{"1", "2"}, nil),
	}

	client := buildClient(skus, nil)

	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, zones)
}

func TestAvailableZones_RestrictionsFiltered(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, []string{"1"}),
	}

	client := buildClient(skus, nil)

	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"2", "3"}, zones)
}

func TestAvailableZones_AllZonesRestricted(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, []string{"1", "2", "3"}),
	}

	client := buildClient(skus, nil)

	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.Empty(t, zones)
}

func TestAvailableZones_MachineNotInRegion(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D8s_v5", []string{"1", "2"}, nil),
	}

	client := buildClient(skus, nil)

	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.Empty(t, zones)
}

func TestAvailableZones_APIError(t *testing.T) {
	client := buildClient(nil, assert.AnError)

	_, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.Error(t, err)
}

func TestAvailableZones_CacheHit(t *testing.T) {
	callCount := 0
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
	}

	mockAPI := &countingMockSKUsAPI{skus: skus, callCounter: &callCount}
	spec := buildProviderSpec([]string{"Standard_D4s_v5"})
	client := &AzureClient{
		skusClient:   mockAPI,
		region:       "westeurope",
		providerSpec: spec,
	}

	zones, err := client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, zones)

	zones, err = client.AvailableZones(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, zones)

	assert.Equal(t, 1, callCount, "API should be called only once due to cache")
}

func TestAvailableZonesCount(t *testing.T) {
	skus := []*armcompute.ResourceSKU{
		buildSKU("Standard_D4s_v5", []string{"1", "2", "3"}, nil),
	}

	client := buildClient(skus, nil)

	count, err := client.AvailableZonesCount(context.Background(), "Standard_D4s_v5")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestAzureClient_RetryOnFillCacheError(t *testing.T) {
	// AzureClient.ensureZonesLoaded retries exactly 'retries' times on persistent API error.
	spec := buildProviderSpec([]string{"Standard_D4s_v5"})
	callCount := 0

	client := &AzureClient{
		skusClient: &countingMockSKUsAPI{
			callCounter: &callCount,
			err:         assert.AnError,
		},
		region:       "westeurope",
		providerSpec: spec,
	}

	err := client.ensureZonesLoaded(context.Background())
	require.Error(t, err)
	assert.Equal(t, retries, callCount)
}

// --- helpers ---

func buildSecret(plainValues map[string]string) *unstructured.Unstructured {
	data := make(map[string]interface{})
	for k, v := range plainValues {
		data[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	s := &unstructured.Unstructured{}
	s.Object = map[string]interface{}{"data": data}
	return s
}

func buildClient(skus []*armcompute.ResourceSKU, apiErr error) *AzureClient {
	// Extract machine names from SKUs so providerSpec knows which ones to keep in cache.
	machineNames := make([]string, 0, len(skus))
	for _, sku := range skus {
		if sku.Name != nil {
			machineNames = append(machineNames, *sku.Name)
		}
	}
	spec := buildProviderSpec(machineNames)
	return &AzureClient{
		skusClient:   &mockSKUsAPI{skus: skus, err: apiErr},
		region:       "westeurope",
		providerSpec: spec,
	}
}

func buildProviderSpec(machineNames []string) *configuration.ProviderSpec {
	machines := ""
	for _, name := range machineNames {
		machines += fmt.Sprintf("    %q: %q\n", name, name)
	}
	yaml := fmt.Sprintf("azure:\n  zonesDiscovery: true\n  machines:\n%s", machines)
	spec, err := configuration.NewProviderSpec(strings.NewReader(yaml))
	if err != nil {
		panic(fmt.Sprintf("buildProviderSpec: failed to parse spec: %v", err))
	}
	return spec
}

func buildSKU(name string, zones []string, restrictedZones []string) *armcompute.ResourceSKU {
	resType := "virtualMachines"
	zonesPtrs := make([]*string, len(zones))
	for i, z := range zones {
		zonesPtrs[i] = &z
	}

	sku := &armcompute.ResourceSKU{
		Name:         &name,
		ResourceType: &resType,
		LocationInfo: []*armcompute.ResourceSKULocationInfo{
			{Zones: zonesPtrs},
		},
	}

	if len(restrictedZones) > 0 {
		restrictedPtrs := make([]*string, len(restrictedZones))
		for i, z := range restrictedZones {
			z := z
			restrictedPtrs[i] = &z
		}
		zoneType := armcompute.ResourceSKURestrictionsTypeZone
		sku.Restrictions = []*armcompute.ResourceSKURestrictions{
			{
				Type: &zoneType,
				RestrictionInfo: &armcompute.ResourceSKURestrictionInfo{
					Zones: restrictedPtrs,
				},
			},
		}
	}

	return sku
}

type mockSKUsAPI struct {
	skus []*armcompute.ResourceSKU
	err  error
}

func (m *mockSKUsAPI) NewListPager(_ *armcompute.ResourceSKUsClientListOptions) *runtime.Pager[armcompute.ResourceSKUsClientListResponse] {
	called := false
	return runtime.NewPager(runtime.PagingHandler[armcompute.ResourceSKUsClientListResponse]{
		More: func(page armcompute.ResourceSKUsClientListResponse) bool {
			return !called
		},
		Fetcher: func(ctx context.Context, _ *armcompute.ResourceSKUsClientListResponse) (armcompute.ResourceSKUsClientListResponse, error) {
			called = true
			if m.err != nil {
				return armcompute.ResourceSKUsClientListResponse{}, m.err
			}
			return armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: m.skus},
			}, nil
		},
	})
}

type countingMockSKUsAPI struct {
	skus        []*armcompute.ResourceSKU
	callCounter *int
	err         error
}

func (m *countingMockSKUsAPI) NewListPager(_ *armcompute.ResourceSKUsClientListOptions) *runtime.Pager[armcompute.ResourceSKUsClientListResponse] {
	called := false
	return runtime.NewPager(runtime.PagingHandler[armcompute.ResourceSKUsClientListResponse]{
		More: func(page armcompute.ResourceSKUsClientListResponse) bool {
			return !called
		},
		Fetcher: func(ctx context.Context, _ *armcompute.ResourceSKUsClientListResponse) (armcompute.ResourceSKUsClientListResponse, error) {
			called = true
			*m.callCounter++
			if m.err != nil {
				return armcompute.ResourceSKUsClientListResponse{}, m.err
			}
			return armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: m.skus},
			}, nil
		},
	})
}

type callbackMockSKUsAPI struct {
	callback func() ([]*armcompute.ResourceSKU, error)
}

func (m *callbackMockSKUsAPI) NewListPager(_ *armcompute.ResourceSKUsClientListOptions) *runtime.Pager[armcompute.ResourceSKUsClientListResponse] {
	called := false
	return runtime.NewPager(runtime.PagingHandler[armcompute.ResourceSKUsClientListResponse]{
		More: func(page armcompute.ResourceSKUsClientListResponse) bool {
			return !called
		},
		Fetcher: func(ctx context.Context, _ *armcompute.ResourceSKUsClientListResponse) (armcompute.ResourceSKUsClientListResponse, error) {
			called = true
			skus, err := m.callback()
			if err != nil {
				return armcompute.ResourceSKUsClientListResponse{}, err
			}
			return armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: skus},
			}, nil
		},
	})
}
