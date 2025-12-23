package provider

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

func TestSapConvergedCloudDefaults(t *testing.T) {

	// given
	sapCC := SapConvergedCloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-eu20",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c", "d"}),
	}

	// when
	values := sapCC.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"eu-de-1a", "eu-de-1b", "eu-de-1c", "eu-de-1d"},
		ProviderType:         "openstack",
		Region:               "eu-de-1",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestSapConvergedCloudTwoZonesRegion(t *testing.T) {

	// given
	region := "eu-de-2"
	sapCC := SapConvergedCloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "cf-eu20",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b"}),
	}

	// when
	values := sapCC.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           2,
		Zones:                []string{"eu-de-2a", "eu-de-2b"},
		ProviderType:         "openstack",
		Region:               "eu-de-2",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestSapConvergedCloudSingleZoneRegion(t *testing.T) {

	// given
	region := "ap-jp-1"
	sapCC := SapConvergedCloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "cf-eu20",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a"}),
	}

	// when
	values := sapCC.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           1,
		Zones:                []string{"ap-jp-1a"},
		ProviderType:         "openstack",
		Region:               "ap-jp-1",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}
