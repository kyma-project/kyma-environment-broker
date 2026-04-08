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

func TestPlanConfigurationWithBlocklist(t *testing.T) {
	spec, err := NewPlanSpecifications(strings.NewReader(`
plan1,plan2:
        operationBlocklist:
            provision: '"provisioning is blocked for this plan","GA=id","owner=team-alpha"'
            update: '"update is blocked for this plan","GA=id2"'
            planUpgrade: '"plan upgrade is blocked for this plan"'
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

	for _, planName := range []string{"plan1", "plan2"} {
		t.Run(planName, func(t *testing.T) {
			bl := spec.OperationBlocklist(planName)
			require.NotNil(t, bl)

			assert.Equal(t, "provisioning is blocked for this plan", bl.Provision.Message)
			assert.Equal(t, map[string]string{"GA": "id", "owner": "team-alpha"}, bl.Provision.Attributes)

			assert.Equal(t, "update is blocked for this plan", bl.Update.Message)
			assert.Equal(t, map[string]string{"GA": "id2"}, bl.Update.Attributes)

			assert.Equal(t, "plan upgrade is blocked for this plan", bl.PlanUpgrade.Message)
			assert.Nil(t, bl.PlanUpgrade.Attributes)
		})
	}

	assert.Nil(t, spec.OperationBlocklist("plan3"))
	assert.Nil(t, spec.OperationBlocklist("sap-converged-cloud"))
	assert.Nil(t, spec.OperationBlocklist("non-existing-plan"))
}

func TestPlanConfigurationWithBlocklist_InlineCommentIsIgnored(t *testing.T) {
	spec, err := NewPlanSpecifications(strings.NewReader(`
plan1:
        operationBlocklist:
            provision: '"provisioning is blocked","GA=id" # this is a comment'
            update: '"update is blocked","GA=id2" # another comment'
            planUpgrade: '"plan upgrade is blocked" # yet another comment'
        regions:
            default:
                - eu-central-1
`))
	require.NoError(t, err)

	bl := spec.OperationBlocklist("plan1")
	require.NotNil(t, bl)

	assert.Equal(t, "provisioning is blocked", bl.Provision.Message)
	assert.Equal(t, map[string]string{"GA": "id"}, bl.Provision.Attributes)

	assert.Equal(t, "update is blocked", bl.Update.Message)
	assert.Equal(t, map[string]string{"GA": "id2"}, bl.Update.Attributes)

	assert.Equal(t, "plan upgrade is blocked", bl.PlanUpgrade.Message)
	assert.Nil(t, bl.PlanUpgrade.Attributes)
}

func TestPlanConfigurationWithBlocklist_MissingMessage(t *testing.T) {
	for _, entry := range []string{`'""'`, `''`} {
		t.Run(entry, func(t *testing.T) {
			_, err := NewPlanSpecifications(strings.NewReader(`
plan1:
        operationBlocklist:
            provision: ` + entry + `
        regions:
            default:
                - eu-central-1
`))
			assert.Error(t, err)
		})
	}
}
