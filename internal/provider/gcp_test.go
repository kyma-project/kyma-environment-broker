package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

func TestGCPDefaults(t *testing.T) {

	// given
	provider := GCPInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-2",
		Region:               "europe-west3",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "pd-balanced",
	}, values)
}

func TestGCPSpecific(t *testing.T) {

	// given
	provider := GCPInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String("us-central1"),
			},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		// default values does not depend on provisioning parameters
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"us-central1-a", "us-central1-b", "us-central1-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-2",
		Region:               "us-central1",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "pd-balanced",
	}, values)
}

func TestGCPTrial_Defaults(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{Region: nil},
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "europe-west3",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_AbstractRegion(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{Region: ptr.String("us")},
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"us-central1-a", "us-central1-b", "us-central1-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "us-central1",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_PlatformRegion(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-us10",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"us-central1-a", "us-central1-b", "us-central1-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "us-central1",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_PlatformRegionNotInMapping(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: nil},
			PlatformRegion: "cf-us11",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "europe-west3",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_PlatformRegionNotInMapping_AbstractRegion(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String("us")},
			PlatformRegion: "cf-us11",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"us-central1-a", "us-central1-b", "us-central1-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "us-central1",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_InvalidAbstractRegion(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{Region: ptr.String("usa")},
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"europe-west3-a", "europe-west3-b", "europe-west3-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "europe-west3",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_RegionNotConsistentWithPlatformRegion(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String("eu-central-1"),
			},
			PlatformRegion: "cf-ap21",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"asia-south1-a", "asia-south1-b", "asia-south1-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "asia-south1",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}

func TestGCPTrial_KSA(t *testing.T) {

	// given
	provider := GCPTrialInputProvider{
		PlatformRegionMapping: TestTrialPlatformRegionMapping,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String("eu-central-1"),
			},
			PlatformRegion: "cf-sa30",
		},
	}

	// when
	values := provider.Provide()

	// then

	assertValues(t, Values{
		// default values do not depend on provisioning parameters
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                []string{"me-central2-a", "me-central2-b", "me-central2-c"},
		ProviderType:         "gcp",
		DefaultMachineType:   "n2-standard-4",
		Region:               "me-central2",
		Purpose:              "evaluation",
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
	}, values)
}
