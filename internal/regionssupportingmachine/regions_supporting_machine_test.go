package regionssupportingmachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRegionsSupportingMachineFromFile(t *testing.T) {
	// given/when
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	// then
	assert.Len(t, regionsSupportingMachine, 3)
	assert.Len(t, regionsSupportingMachine["m8g"], 3)
	assert.Len(t, regionsSupportingMachine["c2d-highmem"], 2)
	assert.Len(t, regionsSupportingMachine["Standard_L"], 3)
}

func TestIsSupported(t *testing.T) {
	// given
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
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
			result := IsSupported(regionsSupportingMachine, tt.region, tt.machineType)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	// given
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name        string
		machineType string
		expected    []string
	}{
		{"Supported m8g", "m8g.large", []string{"ap-northeast-1", "ap-southeast-1", "ca-central-1"}},
		{"Supported c2d-highmem", "c2d-highmem-32", []string{"us-central1", "southamerica-east1"}},
		{"Supported Standard_L", "Standard_L8s_v3", []string{"uksouth", "japaneast", "brazilsouth"}},
		{"Unknown machine type", "unknown-type", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result := SupportedRegions(regionsSupportingMachine, tt.machineType)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}
