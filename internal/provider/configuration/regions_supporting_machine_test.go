package configuration_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRegionsSupportingMachineFromFile(t *testing.T) {
	// given/when
	regionsSupportingMachine, err := configuration.ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	// then
	assert.Len(t, regionsSupportingMachine, 3)

	assert.Len(t, regionsSupportingMachine["m8g"], 3)
	assert.Len(t, regionsSupportingMachine["m8g"]["ap-northeast-1"], 4)
	assert.Len(t, regionsSupportingMachine["m8g"]["ap-southeast-1"], 0)
	assert.Nil(t, regionsSupportingMachine["m8g"]["ca-central-1"])

	assert.Len(t, regionsSupportingMachine["c2d-highmem"], 2)
	assert.Nil(t, regionsSupportingMachine["c2d-highmem"]["us-central1"])
	assert.Len(t, regionsSupportingMachine["c2d-highmem"]["southamerica-east1"], 3)

	assert.Len(t, regionsSupportingMachine["Standard_L"], 3)
	assert.Len(t, regionsSupportingMachine["Standard_L"]["uksouth"], 1)
	assert.Nil(t, regionsSupportingMachine["Standard_L"]["japaneast"])
	assert.Len(t, regionsSupportingMachine["Standard_L"]["brazilsouth"], 2)
}

func TestIsSupported(t *testing.T) {
	// given
	regionsSupportingMachine, err := configuration.ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name        string
		region      string
		machineType string
		expected    bool
	}{
		{"Supported m8g", "ap-northeast-1", "m8g.large", true},
		{"Unsupported m8g", "us-central1", "m8g.2xlarge", false},
		{"Supported c2d-highmem", "us-central1", "c2d-highmem-32", true},
		{"Unsupported c2d-highmem", "ap-southeast-1", "c2d-highmem-64", false},
		{"Supported Standard_L", "uksouth", "Standard_L8s_v3", true},
		{"Unsupported Standard_L", "us-west", "Standard_L48s_v3", false},
		{"Unknown machine type defaults to true", "any-region", "unknown-type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result := regionsSupportingMachine.IsSupported(tt.region, tt.machineType)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	// given
	regionsSupportingMachine, err := configuration.ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name        string
		machineType string
		expected    []string
	}{
		{"Supported m8g", "m8g.large", []string{"ap-northeast-1", "ap-southeast-1", "ca-central-1"}},
		{"Supported c2d-highmem", "c2d-highmem-32", []string{"southamerica-east1", "us-central1"}},
		{"Supported Standard_L", "Standard_L8s_v3", []string{"brazilsouth", "japaneast", "uksouth"}},
		{"Unknown machine type", "unknown-type", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result := regionsSupportingMachine.SupportedRegions(tt.machineType)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAvailableZones(t *testing.T) {
	// given
	regionsSupportingMachine, err := configuration.ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name        string
		provider    string
		machineType string
		region      string
		expected    []string
	}{
		{
			name:        "AWS plan - region with 3 zones",
			provider:    provider.AWSProviderType,
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "AWS plan - region with 2 zones",
			provider:    provider.AWSProviderType,
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"a", "b"},
		},
		{
			name:        "AWS plan - region with 1 zones",
			provider:    provider.AWSProviderType,
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"a"},
		},
		{
			name:        "Azure plan - region with 3 zones",
			provider:    provider.AzureProviderType,
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "Azure plan - region with 2 zones",
			provider:    provider.AzureProviderType,
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"a", "b"},
		},
		{
			name:        "Azure plan - region with 1 zones",
			provider:    provider.AzureProviderType,
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"a"},
		},
		{
			name:        "GCP plan - region with 3 zones",
			provider:    provider.GCPProviderType,
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "GCP plan - region with 2 zones",
			provider:    provider.GCPProviderType,
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"a", "b"},
		},
		{
			name:        "GCP plan - region with 1 zones",
			provider:    provider.GCPProviderType,
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"a"},
		},
		{
			name:        "region with empty list of zones",
			provider:    provider.AWSProviderType,
			machineType: "m8g.large",
			region:      "ap-southeast-1",
			expected:    []string{},
		},
		{
			name:        "region with nil zones",
			provider:    provider.AzureProviderType,
			machineType: "c2d-highmem-32",
			region:      "us-central1",
			expected:    []string{},
		},
		{
			name:        "not supported region",
			provider:    provider.AzureProviderType,
			machineType: "Standard_L8s_v3",
			region:      "westus2",
			expected:    []string{},
		},
		{
			name:        "not supported machine type",
			provider:    provider.AzureProviderType,
			machineType: "notSupportedMachineType",
			region:      "notSupportedRegion",
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result, err := regionsSupportingMachine.AvailableZonesForAdditionalWorkers(tt.machineType, tt.region, tt.provider)
			assert.NoError(t, err)

			// then
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestAvailableZonesForRegionWith4Zones(t *testing.T) {
	// given
	regionsSupportingMachine, err := configuration.ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name     string
		provider string
		expected []string
	}{
		{
			name:     "AWS plan",
			provider: provider.AWSProviderType,
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "Azure plan",
			provider: provider.AzureProviderType,
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "GCP plan",
			provider: provider.GCPProviderType,
			expected: []string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//when
			result, err := regionsSupportingMachine.AvailableZonesForAdditionalWorkers("m8g", "ap-northeast-1", tt.provider)
			assert.NoError(t, err)

			//then
			assert.Len(t, result, len(tt.expected))
			for _, v := range result {
				assert.Contains(t, tt.expected, v)
			}
		})
	}
}
