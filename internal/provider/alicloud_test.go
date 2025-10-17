package provider

import (
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
