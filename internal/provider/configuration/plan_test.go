package configuration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanConfiguration(t *testing.T) {
	// given
	spec, err := NewPlanSpecifications(strings.NewReader(`
plan1,plan2:
        regions:
            cf-eu11:
                - eu-central-1
                - eu-west-2
            default:
                - eu-central-1
                - eu-west-1
                - us-east-1
plan3:
        regions:
            cf-eu11:
                - westeurope
            default:
                - japaneast
                - easteurope
sap-converged-cloud:
      regions:
        cf-eu20:
          - "eu-de-1"
`))
	require.NoError(t, err)

	// when / then

	assert.Equal(t, []string{"eu-central-1", "eu-west-2"}, spec.Regions("plan1", "cf-eu11"))
	assert.Equal(t, []string{"eu-central-1", "eu-west-2"}, spec.Regions("plan2", "cf-eu11"))
	assert.Equal(t, []string{"westeurope"}, spec.Regions("plan3", "cf-eu11"))

	// take default regions
	assert.Equal(t, []string{"eu-central-1", "eu-west-1", "us-east-1"}, spec.Regions("plan1", "cf-us11"))
	assert.Equal(t, []string{"eu-central-1", "eu-west-1", "us-east-1"}, spec.Regions("plan2", "cf-us11"))
	assert.Equal(t, []string{"japaneast", "easteurope"}, spec.Regions("plan3", "cf-us11"))
}
