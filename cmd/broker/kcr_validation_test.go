package main

import (
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvedMachineTypesForKCR_ResolvesAliases(t *testing.T) {
	// Aliases like "ri.8xlarge" must be resolved to their real type (e.g. "r7i.8xlarge")
	// before being passed to ValidateAllMachineTypes, because the KCR ConfigMap is keyed
	// by the real/resolved type, not the customer-facing alias.
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "ri.8xlarge": "ri.8xlarge display"
    "mi.4xlarge": "mi.4xlarge display"
    "m6i.large":  "m6i.large display"
  machinesVersions:
    "ri.{size}": "r7i.{size}"
    "mi.{size}": "m7i.{size}"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{broker.AWSPlanName})

	awsTypes := result["AWS"]
	assert.Contains(t, awsTypes, "r7i.8xlarge", "alias ri.8xlarge must be resolved to r7i.8xlarge")
	assert.Contains(t, awsTypes, "m7i.4xlarge", "alias mi.4xlarge must be resolved to m7i.4xlarge")
	assert.Contains(t, awsTypes, "m6i.large", "type without alias must pass through unchanged")
	assert.NotContains(t, awsTypes, "ri.8xlarge", "raw alias must not appear in result")
	assert.NotContains(t, awsTypes, "mi.4xlarge", "raw alias must not appear in result")
}

func TestResolvedMachineTypesForKCR_DeduplicatesResolvedTypes(t *testing.T) {
	// Two aliases that collapse to the same resolved type must only appear once.
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "alias-a.large": "display a"
    "alias-b.large": "display b"
  machinesVersions:
    "alias-a.{size}": "real.{size}"
    "alias-b.{size}": "real.{size}"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{broker.AWSPlanName})

	count := 0
	for _, mt := range result["AWS"] {
		if mt == "real.large" {
			count++
		}
	}
	assert.Equal(t, 1, count, "deduplicated: real.large must appear exactly once")
}

func TestResolvedMachineTypesForKCR_DisabledPlanExcludesProvider(t *testing.T) {
	// If the gdch plan is not in EnablePlans, GDCH machines must not appear in the result.
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large display"
gdch:
  machines:
    "n3-standard-2-gdc": "n3-standard-2-gdc display"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{broker.AWSPlanName})

	assert.NotEmpty(t, result["AWS"])
	assert.Empty(t, result["GDCH"], "GDCH machines must not be validated when gdch plan is disabled")
}

func TestResolvedMachineTypesForKCR_EnabledGDCHPlanIncludesGDCHProvider(t *testing.T) {
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large display"
gdch:
  machines:
    "n3-standard-2-gdc": "n3-standard-2-gdc display"
    "n3-standard-4-gdc": "n3-standard-4-gdc display"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{broker.AWSPlanName, broker.GDCHPlanName})

	assert.NotEmpty(t, result["AWS"])
	assert.ElementsMatch(t, []string{"n3-standard-2-gdc", "n3-standard-4-gdc"}, result["GDCH"])
}

func TestResolvedMachineTypesForKCR_MultipleAWSPlansDeduplicateProvider(t *testing.T) {
	// aws and build-runtime-aws both map to AWS — machines must not be duplicated.
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large display"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{broker.AWSPlanName, broker.BuildRuntimeAWSPlanName})

	assert.Equal(t, []string{"m6i.large"}, result["AWS"], "same machine must appear exactly once even with two AWS plans enabled")
}

func TestResolvedMachineTypesForKCR_UnknownPlanIsIgnored(t *testing.T) {
	// Plans that don't map to a hyperscaler (trial, free) must not cause errors or add providers.
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large display"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{broker.AWSPlanName, broker.TrialPlanName, broker.FreemiumPlanName})

	assert.Len(t, result, 1, "only AWS provider must be present; trial and free must be ignored")
	assert.NotEmpty(t, result["AWS"])
}

func TestResolvedMachineTypesForKCR_EmptyEnabledPlansReturnsEmpty(t *testing.T) {
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large display"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, broker.StringList{})

	assert.Empty(t, result)
}
