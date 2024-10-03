package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

var AzureTrialPlatformRegionMapping = map[string]string{"cf-eu11": "europe", "cf-ap21": "asia"}

func TestAzureDefaults(t *testing.T) {

	// given
	azure := AzureInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D2s_v5",
		Region:               "eastus",
		Purpose:              "production",
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
	}, values)
}

func TestAzureTrialDefaults(t *testing.T) {

	// given
	azure := AzureTrialInputProvider{
		PlatformRegionMapping: AzureTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D4s_v5",
		Region:               "switzerlandnorth",
		Purpose:              "evaluation",
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
	}, values)
}

func TestAzureLiteDefaults(t *testing.T) {

	// given
	azure := AzureLiteInputProvider{
		Purpose: PurposeEvaluation,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 2,
		ZonesCount:           1,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D4s_v5",
		Region:               "eastus",
		Purpose:              "evaluation",
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
	}, values)
}

func TestAzureSpecific(t *testing.T) {

	// given
	azure := AzureInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-eu11",
			PlatformProvider: "azure",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D2s_v5",
		Region:               "uksouth",
		Purpose:              "production",
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
	}, values)
}

func TestAzureTrialSpecific(t *testing.T) {

	// given
	azure := AzureTrialInputProvider{
		PlatformRegionMapping: AzureTrialPlatformRegionMapping,

		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-ap21",
			PlatformProvider: "azure",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D4s_v5",
		Region:               "southeastasia",
		Purpose:              "evaluation",
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
	}, values)
}

func TestAzureLiteSpecific(t *testing.T) {

	// given
	azure := AzureLiteInputProvider{
		Purpose: PurposeEvaluation,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-eu11",
			PlatformProvider: "azure",
		},
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, Values{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 2,
		ZonesCount:           1,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		DefaultMachineType:   "Standard_D4s_v5",
		Region:               "uksouth",
		Purpose:              "evaluation",
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
	}, values)
}
