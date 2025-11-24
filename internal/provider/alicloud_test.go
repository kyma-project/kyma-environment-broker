package provider

import (
	"strings"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

func TestAlicloudDefaults(t *testing.T) {

	// given
	alicloud := AlicloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "eu-central-1",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := alicloud.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
		ProviderType:         "alicloud",
		DefaultMachineType:   "ecs.g8i.large",
		Region:               "eu-central-1",
		Purpose:              "production",
		DiskType:             DefaultAlicloudDiskType,
		VolumeSizeGb:         80,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAlicloudTwoZonesRegion(t *testing.T) {

	// given
	region := "eu-central-1"
	alicloud := AlicloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "eu-central-1",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b"}),
	}

	// when
	values := alicloud.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           2,
		Zones:                []string{"eu-central-1a", "eu-central-1b"},
		ProviderType:         "alicloud",
		DefaultMachineType:   "ecs.g8i.large",
		Region:               "eu-central-1",
		Purpose:              "production",
		DiskType:             DefaultAlicloudDiskType,
		VolumeSizeGb:         80,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAlicloudSingleZoneRegion(t *testing.T) {

	// given
	region := "eu-central-1"
	alicloud := AlicloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "eu-central-1",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a"}),
	}

	// when
	values := alicloud.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           1,
		Zones:                []string{"eu-central-1a"},
		ProviderType:         "alicloud",
		DefaultMachineType:   "ecs.g8i.large",
		Region:               "eu-central-1",
		Purpose:              "production",
		DiskType:             DefaultAlicloudDiskType,
		VolumeSizeGb:         80,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAlicloudInputProvider_MultipleProvisionsDoNotCorruptZones(t *testing.T) {
	// given
	// Create SHARED ZonesProvider (simulating production behavior)
	zonesProvider := FakeZonesProvider([]string{"a", "b", "c"})

	// Create provider that will be reused (simulating production)
	createProvider := func() *AlicloudInputProvider {
		return &AlicloudInputProvider{
			MultiZone:     true,
			Purpose:       PurposeProduction,
			ZonesProvider: zonesProvider, // SHARED across all calls
			ProvisioningParameters: internal.ProvisioningParameters{
				Parameters: pkg.ProvisioningParametersDTO{
					Region: ptr.String("eu-central-1"),
				},
			},
			FailureTolerance: "zone",
		}
	}

	// when - simulate multiple provisions (like in production/e2e)
	var results []internal.ProviderValues
	for i := 0; i < 5; i++ {
		provider := createProvider()
		result := provider.Provide()
		results = append(results, result)

		t.Logf("Iteration %d: zones = %v", i+1, result.Zones)
	}

	// then - all results should have correctly formatted zones
	expectedZones := []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}

	for i, result := range results {
		if len(result.Zones) != 3 {
			t.Errorf("Iteration %d should have 3 zones, got %d", i+1, len(result.Zones))
		}

		// Check each zone has correct format (not duplicated regions)
		for j, zone := range result.Zones {
			// Zone should be exactly "eu-central-1" + one letter
			if !isValidAlicloudZoneFormat(zone) {
				t.Errorf("Iteration %d, zone %d should be 'eu-central-1' + single letter, got: %s", i+1, j, zone)
			}

			// More specific: should not contain duplicate regions
			if strings.Contains(zone, "eu-central-1eu-central-1") {
				t.Errorf("Iteration %d, zone %d contains duplicated region: %s", i+1, j, zone)
			}
		}

		// Zones should contain all expected zones (order may vary due to shuffle)
		if !containsAllZones(result.Zones, expectedZones) {
			t.Errorf("Iteration %d zones %v should match expected zones %v", i+1, result.Zones, expectedZones)
		}
	}
}

func TestAlicloudInputProvider_ZonesNotSharedBetweenCalls(t *testing.T) {
	// given
	zonesProvider := FakeZonesProvider([]string{"a", "b", "c"})

	provider := &AlicloudInputProvider{
		MultiZone:     true,
		Purpose:       PurposeProduction,
		ZonesProvider: zonesProvider,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Region: ptr.String("eu-central-1"),
			},
		},
		FailureTolerance: "zone",
	}

	// when - call Provide() multiple times
	firstCall := provider.Provide()
	secondCall := provider.Provide()
	thirdCall := provider.Provide()

	// then - verify zones are correctly formatted in all calls
	validateZones := func(zones []string, callNumber int) {
		for i, zone := range zones {
			// Should match pattern exactly once
			matches := strings.Count(zone, "eu-central-1")
			if matches != 1 {
				t.Errorf("Call %d: zone[%d]=%s should contain 'eu-central-1' exactly once, found %d times",
					callNumber, i, zone, matches)
			}

			// Should be valid zone format
			if !isValidAlicloudZoneFormat(zone) {
				t.Errorf("Call %d: zone[%d]=%s should match format 'eu-central-1[a-c]'",
					callNumber, i, zone)
			}
		}
	}

	validateZones(firstCall.Zones, 1)
	validateZones(secondCall.Zones, 2)
	validateZones(thirdCall.Zones, 3)
}

// Helper functions for the new tests
func isValidAlicloudZoneFormat(zone string) bool {
	// Zone should be "eu-central-1" followed by a single letter
	if !strings.HasPrefix(zone, "eu-central-1") {
		return false
	}
	suffix := strings.TrimPrefix(zone, "eu-central-1")
	// Suffix should be exactly one letter
	return len(suffix) == 1 && suffix >= "a" && suffix <= "z"
}

func containsAllZones(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	expectedMap := make(map[string]bool)
	for _, zone := range expected {
		expectedMap[zone] = true
	}
	for _, zone := range actual {
		if !expectedMap[zone] {
			return false
		}
	}
	return true
}
