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

func TestAlicloudInputProvider_DifferentRegionFormats(t *testing.T) {
	tests := []struct {
		name           string
		region         string
		configZones    []string
		expectedFormat string // regex pattern
		description    string
	}{
		{
			name:           "numeric suffix region uses direct concatenation",
			region:         "eu-central-1",
			configZones:    []string{"a", "b", "c"},
			expectedFormat: `^eu-central-1[a-c]$`,
			description:    "Regions ending in digit: eu-central-1 + a = eu-central-1a",
		},
		{
			name:           "alpha suffix region uses hyphen separator",
			region:         "cn-beijing",
			configZones:    []string{"a", "b", "c"},
			expectedFormat: `^cn-beijing-[a-c]$`,
			description:    "Regions ending in letter: cn-beijing + - + a = cn-beijing-a",
		},
		{
			name:           "shanghai region uses hyphen",
			region:         "cn-shanghai",
			configZones:    []string{"a", "b", "c", "d"},
			expectedFormat: `^cn-shanghai-[a-d]$`,
			description:    "cn-shanghai + - + a = cn-shanghai-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			zonesProvider := FakeZonesProvider(tt.configZones)

			provider := &AlicloudInputProvider{
				MultiZone:     true,
				Purpose:       PurposeProduction,
				ZonesProvider: zonesProvider,
				ProvisioningParameters: internal.ProvisioningParameters{
					Parameters: pkg.ProvisioningParametersDTO{
						Region: ptr.String(tt.region),
					},
				},
				FailureTolerance: "zone",
			}

			// when - call multiple times to ensure no corruption
			var results []internal.ProviderValues
			for i := 0; i < 3; i++ {
				result := provider.Provide()
				results = append(results, result)
			}

			// then - all iterations should produce correctly formatted zones
			for iteration, result := range results {
				t.Logf("Iteration %d - %s: zones = %v", iteration+1, tt.description, result.Zones)

				for i, zone := range result.Zones {
					// Should match expected format
					matched := matchesPattern(zone, tt.expectedFormat)
					if !matched {
						t.Errorf("Iteration %d: Zone[%d]=%s should match pattern %s (%s)",
							iteration+1, i, zone, tt.expectedFormat, tt.description)
					}

					// Should not contain duplicated region names
					regionCount := strings.Count(zone, tt.region)
					if regionCount != 1 {
						t.Errorf("Iteration %d: Zone[%d]=%s contains region '%s' %d times, expected 1",
							iteration+1, i, zone, tt.region, regionCount)
					}
				}
			}
		})
	}
}

func matchesPattern(s, pattern string) bool {
	// Simple pattern matching for the test cases
	if pattern == `^eu-central-1[a-c]$` {
		return len(s) == len("eu-central-1a") && strings.HasPrefix(s, "eu-central-1") && s[len(s)-1] >= 'a' && s[len(s)-1] <= 'c'
	}
	if pattern == `^cn-beijing-[a-c]$` {
		return len(s) == len("cn-beijing-a") && strings.HasPrefix(s, "cn-beijing-") && s[len(s)-1] >= 'a' && s[len(s)-1] <= 'c'
	}
	if pattern == `^cn-shanghai-[a-d]$` {
		return len(s) == len("cn-shanghai-a") && strings.HasPrefix(s, "cn-shanghai-") && s[len(s)-1] >= 'a' && s[len(s)-1] <= 'd'
	}
	return false
}
