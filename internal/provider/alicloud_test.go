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
			PlatformRegion: "cn-beijing",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c", "d"}),
	}

	// when
	values := alicloud.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"cn-beijing-a", "cn-beijing-b", "cn-beijing-c", "cn-beijing-d"},
		ProviderType:         "alicloud",
		DefaultMachineType:   "ecs.g6.large",
		Region:               "cn-beijing",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAlicloudTwoZonesRegion(t *testing.T) {

	// given
	region := "cn-shanghai"
	alicloud := AlicloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "cn-beijing",
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
		Zones:                []string{"cn-shanghai-a", "cn-shanghai-b"},
		ProviderType:         "alicloud",
		DefaultMachineType:   "ecs.g6.large",
		Region:               "cn-shanghai",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAlicloudSingleZoneRegion(t *testing.T) {

	// given
	region := "cn-hangzhou"
	alicloud := AlicloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "cn-beijing",
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
		Zones:                []string{"cn-hangzhou-a"},
		ProviderType:         "alicloud",
		DefaultMachineType:   "ecs.g6.large",
		Region:               "cn-hangzhou",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}
