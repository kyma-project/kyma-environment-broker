package provider

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
)

var TestTrialPlatformRegionMapping = map[string]string{"cf-eu10": "europe", "cf-us10": "us", "cf-ap21": "asia"}

func TestAWSDefaults(t *testing.T) {

	// given
	provider := AWSInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-eu11",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
		ProviderType:         "aws",
		Region:               "eu-central-1",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "gp3",
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAWSSpecific(t *testing.T) {

	// given
	provider := AWSInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Region: ptr.String("ap-southeast-1"),
			},
			PlatformRegion: "cf-eu11",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		// default values does not depend on provisioning parameters
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"ap-southeast-1a", "ap-southeast-1b", "ap-southeast-1c"},
		ProviderType:         "aws",
		Region:               "ap-southeast-1",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "gp3",
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAWSTrialDefaults(t *testing.T) {

	// given
	provider := AWSTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-eu11",
		},
		ZonesProvider: FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
		ProviderType:         "aws",
		DefaultMachineType:   "m5.xlarge",
		Region:               "eu-central-1",
		Purpose:              "evaluation",
		VolumeSizeGb:         50,
		DiskType:             "gp3",
		FailureTolerance:     nil,
	}, values)
}

func TestAWSTrialSpecific(t *testing.T) {

	// given
	provider := AWSTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Region: ptr.String("eu-central-1"),
			},
			PlatformRegion: "cf-ap21",
		},
		ZonesProvider: FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"ap-southeast-1a", "ap-southeast-1b", "ap-southeast-1c"},
		ProviderType:         "aws",
		DefaultMachineType:   "m5.xlarge",
		Region:               "ap-southeast-1",
		Purpose:              "evaluation",
		VolumeSizeGb:         50,
		DiskType:             "gp3",
		FailureTolerance:     nil,
	}, values)
}

func assertValues(t *testing.T, expected internal.ProviderValues, got internal.ProviderValues) {
	assert.Equal(t, expected.ZonesCount, len(got.Zones))
	assert.Subset(t, expected.Zones, got.Zones)
	got.Zones = nil
	expected.Zones = nil
	assert.Equal(t, expected, got)
}
