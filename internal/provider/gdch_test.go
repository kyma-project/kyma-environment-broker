package provider

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

func TestGDCHDefaults(t *testing.T) {
	// given
	p := GDCHInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: false,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{Region: nil},
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := p.Provide()

	// then
	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 3,
		ZonesCount:           1,
		Zones:                []string{"us-west16-a"},
		ProviderType:         "gdch",
		Region:               "us-west16",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "Standard",
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestGDCHMultiZone(t *testing.T) {
	// given
	p := GDCHInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{Region: nil},
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"a", "b", "c"}),
	}

	// when
	values := p.Provide()

	// then
	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"us-west16-a", "us-west16-b", "us-west16-c"},
		ProviderType:         "gdch",
		Region:               "us-west16",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "Standard",
		FailureTolerance:     ptr.String("zone"),
	}, values)
}

func TestGDCHCustomRegion(t *testing.T) {
	// given
	p := GDCHInputProvider{
		Purpose:   PurposeProduction,
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{Region: ptr.String("us-east1")},
		},
		FailureTolerance: "zone",
		ZonesProvider:    FakeZonesProvider([]string{"b", "c", "d"}),
	}

	// when
	values := p.Provide()

	// then
	assertValues(t, internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"us-east1-b", "us-east1-c", "us-east1-d"},
		ProviderType:         "gdch",
		Region:               "us-east1",
		Purpose:              "production",
		VolumeSizeGb:         80,
		DiskType:             "Standard",
		FailureTolerance:     ptr.String("zone"),
	}, values)
}
