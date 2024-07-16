package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
)

func TestAWSDefaults(t *testing.T) {

	// given
	aws := AWSInputProvider{
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters:     internal.ProvisioningParametersDTO{Region: ptr.String("eu-central-1")},
			PlatformRegion: "cf-eu11",
		},
	}

	// when
	values := aws.Provide()

	// then

	assertValues(t, Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"eu-central-1a", "eu-central-1b", "ap-southeast-1c"},
		ProviderType:         "aws",
		DefaultMachineType:   "m6i.large",
		Region:               "eu-central-1",
		Purpose:              "production",
	}, values)
}

func TestAWSSpecific(t *testing.T) {

	// given
	aws := AWSInputProvider{
		MultiZone: true,
		ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				MachineType: ptr.String("m6i.xlarge"),
				Region:      ptr.String("ap-southeast-1"),
			},
			PlatformRegion:   "cf-eu11",
			PlatformProvider: "ap-southeast-1",
		},
	}

	// when
	values := aws.Provide()

	// then

	assertValues(t, Values{
		// default values does not depend on provisioning parameters
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           3,
		Zones:                []string{"ap-southeast-1a", "ap-southeast-1b", "ap-southeast-1c"},
		ProviderType:         "aws",
		DefaultMachineType:   "m6i.large",
		Region:               "ap-southeast-1",
		Purpose:              "production",
	}, values)
}

func assertValues(t *testing.T, expected Values, got Values) {
	assert.Len(t, expected.Zones, len(got.Zones))
	got.Zones = nil
	expected.Zones = nil
	assert.Equal(t, expected, got)
}
