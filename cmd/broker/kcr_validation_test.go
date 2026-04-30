package main

import (
	"strings"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
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

	result := resolvedMachineTypesForKCR(providerSpec, []pkg.CloudProvider{pkg.AWS})

	awsTypes := result[pkg.AWS]
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

	result := resolvedMachineTypesForKCR(providerSpec, []pkg.CloudProvider{pkg.AWS})

	count := 0
	for _, mt := range result[pkg.AWS] {
		if mt == "real.large" {
			count++
		}
	}
	assert.Equal(t, 1, count, "deduplicated: real.large must appear exactly once")
}

func TestResolvedMachineTypesForKCR_EmptyProviderReturnsEmpty(t *testing.T) {
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large display"
`))
	require.NoError(t, err)

	result := resolvedMachineTypesForKCR(providerSpec, []pkg.CloudProvider{pkg.Azure})

	assert.Empty(t, result[pkg.Azure])
}
