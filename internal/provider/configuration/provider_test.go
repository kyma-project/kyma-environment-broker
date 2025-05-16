package configuration

import (
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSpec(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: [ "a", "b", "f" ]
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
        zones: [ "a", "b", "c" ]
azure:
    regions:
      westeurope:
        displayName: "westeurope (Europe, Netherlands)"
        zones: [ "1", "2", "3" ]
`))
	require.NoError(t, err)

	// when / then

	assert.Equal(t, "eu-central-1 (Europe, Frankfurt)", providerSpec.RegionDisplayName(runtime.AWS, "eu-central-1"))
	assert.Equal(t, []string{"a", "b", "f"}, providerSpec.Zones(runtime.AWS, "eu-central-1"))

	assert.Equal(t, "westeurope (Europe, Netherlands)", providerSpec.RegionDisplayName(runtime.Azure, "westeurope"))
	assert.Equal(t, []string{"1", "2", "3"}, providerSpec.Zones(runtime.Azure, "westeurope"))
}

func TestProviderSpec_NotDefined(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: [ "a", "b", "f" ]
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
        zones: [ "a", "b", "c" ]
azure:
    regions:
      westeurope:
        displayName: "westeurope (Europe, Netherlands)"
        zones: [ "1", "2", "3" ]

`))
	require.NoError(t, err)

	// when / then

	assert.Equal(t, "us-east-1", providerSpec.RegionDisplayName(runtime.AWS, "us-east-1"))
	assert.Equal(t, []string{}, providerSpec.Zones(runtime.AWS, "us-east-1"))
}

func TestProviderSpec_Validation(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
  aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: []
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
      eu-west-1: 
        zones: [ "a", "b", "c" ]
      us-east-1:
        displayName: "us-east-1 (US, Virginia)"
        zones: [ "a", "b", "c" ]
`))
	require.NoError(t, err)

	// when / then

	assert.Errorf(t, providerSpec.Validate(runtime.AWS, "eu-central-1"), "region eu-central-1 for provider aws has no zones defined")
	assert.Errorf(t, providerSpec.Validate(runtime.AWS, "eu-west-2"), "region eu-west-2 for provider aws has no zones defined")
	assert.Errorf(t, providerSpec.Validate(runtime.AWS, "eu-west-1"), "region eu-west-1 for provider aws has no display name defined")
	assert.NoError(t, providerSpec.Validate(runtime.AWS, "us-east-1"))
}
