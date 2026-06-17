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
        upgradableToPlans: [plan3-bis]
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

	// upgradable plans
	assert.True(t, spec.IsUpgradableBetween("plan3", "plan3-bis"))
	assert.False(t, spec.IsUpgradableBetween("plan3", "plan1"))
	assert.False(t, spec.IsUpgradableBetween("plan1", "plan3-bis"))
	assert.False(t, spec.IsUpgradableBetween("plan1-not-existing", "plan2"))
}

func TestWarnInternalOnlyMachines(t *testing.T) {
	t.Run("redundant entry covered by prefix", func(t *testing.T) {
		spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    regularMachines: [g6.xlarge, g6.2xlarge]
    internalOnlyMachines: [g6, g6.xlarge]
`))
		require.NoError(t, err)
		warnings := spec.ValidateInternalOnlyMachines()
		require.Len(t, warnings, 1)
		assert.Equal(t, `internalOnlyMachines entry "g6.xlarge" is redundant in plan "aws" — already covered by prefix "g6"`, warnings[0])
	})

	t.Run("unknown entry matches nothing", func(t *testing.T) {
		spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    regularMachines: [m5.xlarge]
    internalOnlyMachines: [g9]
`))
		require.NoError(t, err)
		warnings := spec.ValidateInternalOnlyMachines()
		require.Len(t, warnings, 1)
		assert.Equal(t, `internalOnlyMachines entry "g9" in plan "aws" does not match any machine type in regularMachines or additionalMachines`, warnings[0])
	})

	t.Run("multiple warnings", func(t *testing.T) {
		spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    regularMachines: [g6.xlarge, g6.2xlarge]
    internalOnlyMachines: [g6, g6.xlarge, g9]
`))
		require.NoError(t, err)
		warnings := spec.ValidateInternalOnlyMachines()
		assert.ElementsMatch(t, []string{
			`internalOnlyMachines entry "g6.xlarge" is redundant in plan "aws" — already covered by prefix "g6"`,
			`internalOnlyMachines entry "g9" in plan "aws" does not match any machine type in regularMachines or additionalMachines`,
		}, warnings)
	})

	t.Run("entry matches additionalMachines", func(t *testing.T) {
		spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    regularMachines: [m5.xlarge]
    additionalMachines: [g6.xlarge]
    internalOnlyMachines: [g6]
`))
		require.NoError(t, err)
		assert.Empty(t, spec.ValidateInternalOnlyMachines())
	})

	t.Run("no warnings for valid config", func(t *testing.T) {
		spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    regularMachines: [m5.xlarge, g4dn.xlarge]
    internalOnlyMachines: [g4dn]
`))
		require.NoError(t, err)
		assert.Empty(t, spec.ValidateInternalOnlyMachines())
	})

	t.Run("no internalOnlyMachines", func(t *testing.T) {
		spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    regularMachines: [m5.xlarge]
`))
		require.NoError(t, err)
		assert.Empty(t, spec.ValidateInternalOnlyMachines())
	})
}

func TestIsInternalOnlyMachine(t *testing.T) {
	spec, err := NewPlanSpecifications(strings.NewReader(`
aws:
    internalOnlyMachines:
        - g4dn.xlarge
        - g6
azure:
`))
	require.NoError(t, err)

	// exact match
	assert.True(t, spec.IsInternalOnlyMachine("aws", "g4dn.xlarge"))
	// prefix match
	assert.True(t, spec.IsInternalOnlyMachine("aws", "g6.xlarge"))
	// no match
	assert.False(t, spec.IsInternalOnlyMachine("aws", "m5.xlarge"))
	assert.False(t, spec.IsInternalOnlyMachine("aws", "g4dn.2xlarge"))
	// unknown plan
	assert.False(t, spec.IsInternalOnlyMachine("unknown", "g6.xlarge"))
	// plan with no internalOnlyMachines
	assert.False(t, spec.IsInternalOnlyMachine("azure", "Standard_NC4as_T4_v3"))
}
