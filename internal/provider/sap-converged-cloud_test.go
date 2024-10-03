package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

func TestSapConvergedCloudDefaults(t *testing.T) {

	// given
	sapCC := SapConvergedCloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-eu20",
		},
	}

	// when
	values := sapCC.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"eu-de-1a", "eu-de-1b", "eu-de-1c", "eu-de-1d"},
		ProviderType:         "openstack",
		DefaultMachineType:   "g_c2_m8",
		Region:               "eu-de-1",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
	}, values)
}

func TestSapConvergedCloudTwoZonesRegion(t *testing.T) {

	// given
	region := "eu-de-2"
	sapCC := SapConvergedCloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "cf-eu20",
		},
	}

	// when
	values := sapCC.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           2,
		Zones:                []string{"eu-de-2a", "eu-de-2b"},
		ProviderType:         "openstack",
		DefaultMachineType:   "g_c2_m8",
		Region:               "eu-de-2",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
	}, values)
}

func TestSapConvergedCloudSingleZoneRegion(t *testing.T) {

	// given
	region := "ap-jp-1"
	sapCC := SapConvergedCloudInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String(region)},
			PlatformRegion: "cf-eu20",
		},
	}

	// when
	values := sapCC.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           1,
		Zones:                []string{"ap-jp-1a"},
		ProviderType:         "openstack",
		DefaultMachineType:   "g_c2_m8",
		Region:               "ap-jp-1",
		Purpose:              "production",
		DiskType:             "",
		VolumeSizeGb:         0,
	}, values)
}
