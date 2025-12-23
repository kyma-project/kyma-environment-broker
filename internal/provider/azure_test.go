package provider

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
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
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"1", "2", "3"}),
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		Region:               "eastus",
		Purpose:              "production",
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAzureTrialDefaults(t *testing.T) {

	// given
	azure := AzureTrialInputProvider{
		PlatformRegionMapping: AzureTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
		ZonesProvider: FakeZonesProvider([]string{"1", "2", "3"}),
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, internal.ProviderValues{
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
		FailureTolerance:     nil,
	}, values)
}

func TestAzureLiteDefaults(t *testing.T) {

	// given
	azure := AzureLiteInputProvider{
		Purpose: PurposeEvaluation,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     pkg.ProvisioningParametersDTO{Region: ptr.String("eastus")},
			PlatformRegion: "cf-eu11",
		},
		ZonesProvider: FakeZonesProvider([]string{"1", "2", "3"}),
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 2,
		ZonesCount:           1,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		Region:               "eastus",
		Purpose:              "evaluation",
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
	}, values)
}

func TestAzureSpecific(t *testing.T) {

	// given
	azure := AzureInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-eu11",
			PlatformProvider: "azure",
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"1", "2", "3"}),
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		Region:               "uksouth",
		Purpose:              "production",
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestAzureTrialSpecific(t *testing.T) {

	// given
	azure := AzureTrialInputProvider{
		PlatformRegionMapping: AzureTrialPlatformRegionMapping,

		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-ap21",
			PlatformProvider: "azure",
		},
		ZonesProvider: FakeZonesProvider([]string{"1", "2", "3"}),
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, internal.ProviderValues{
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
		FailureTolerance:     nil,
	}, values)
}

func TestAzureLiteSpecific(t *testing.T) {

	// given
	azure := AzureLiteInputProvider{
		Purpose: PurposeEvaluation,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				MachineType: ptr.String("Standard_D48_v3"),
				Region:      ptr.String("uksouth"),
			},
			PlatformRegion:   "cf-eu11",
			PlatformProvider: "azure",
		},
		ZonesProvider: FakeZonesProvider([]string{"1", "2", "3"}),
	}

	// when
	values := azure.Provide()

	// then

	assertValues(t, internal.ProviderValues{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 2,
		ZonesCount:           1,
		Zones:                []string{"1", "2", "3"},
		ProviderType:         "azure",
		Region:               "uksouth",
		Purpose:              "evaluation",
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
		FailureTolerance:     nil,
	}, values)
}
