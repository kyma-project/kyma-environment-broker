package regionssupportingmachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRegionsSupportingMachineFromFile(t *testing.T) {
	t.Run("Zone mapping disabled", func(t *testing.T) {
		// given/when
		regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml", false)
		require.NoError(t, err)

		// then
		assert.Len(t, regionsSupportingMachine, 3)

		assert.Len(t, regionsSupportingMachine["m8g"], 3)
		assert.Nil(t, regionsSupportingMachine["m8g"]["ap-northeast-1"])
		assert.Nil(t, regionsSupportingMachine["m8g"]["ap-southeast-1"])
		assert.Nil(t, regionsSupportingMachine["m8g"]["ca-central-1"])

		assert.Len(t, regionsSupportingMachine["c2d-highmem"], 2)
		assert.Nil(t, regionsSupportingMachine["c2d-highmem"]["us-central1"])
		assert.Nil(t, regionsSupportingMachine["c2d-highmem"]["southamerica-east1"])

		assert.Len(t, regionsSupportingMachine["Standard_L"], 3)
		assert.Nil(t, regionsSupportingMachine["Standard_L"]["uksouth"])
		assert.Nil(t, regionsSupportingMachine["Standard_L"]["japaneast"])
		assert.Nil(t, regionsSupportingMachine["Standard_L"]["brazilsouth"])
	})

	t.Run("Zone mapping enabled", func(t *testing.T) {
		// given/when
		regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine-with-zones.yaml", true)
		require.NoError(t, err)

		// then
		assert.Len(t, regionsSupportingMachine, 3)

		assert.Len(t, regionsSupportingMachine["m8g"], 3)
		assert.Len(t, regionsSupportingMachine["m8g"]["ap-northeast-1"], 4)
		assert.Nil(t, regionsSupportingMachine["m8g"]["ap-southeast-1"])
		assert.Nil(t, regionsSupportingMachine["m8g"]["ca-central-1"])

		assert.Len(t, regionsSupportingMachine["c2d-highmem"], 2)
		assert.Nil(t, regionsSupportingMachine["c2d-highmem"]["us-central1"])
		assert.Len(t, regionsSupportingMachine["c2d-highmem"]["southamerica-east1"], 3)

		assert.Len(t, regionsSupportingMachine["Standard_L"], 3)
		assert.Nil(t, regionsSupportingMachine["Standard_L"]["uksouth"])
		assert.Nil(t, regionsSupportingMachine["Standard_L"]["japaneast"])
		assert.Len(t, regionsSupportingMachine["Standard_L"]["brazilsouth"], 2)
	})
}

func TestIsSupported(t *testing.T) {
	for tn, tc := range map[string]struct {
		fileName    string
		zoneMapping bool
	}{
		"Zone mapping disabled": {
			fileName:    "test/regions-supporting-machine.yaml",
			zoneMapping: false,
		},
		"Zone mapping enabled": {
			fileName:    "test/regions-supporting-machine-with-zones.yaml",
			zoneMapping: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile(tc.fileName, tc.zoneMapping)
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
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	for tn, tc := range map[string]struct {
		fileName    string
		zoneMapping bool
	}{
		"Zone mapping disabled": {
			fileName:    "test/regions-supporting-machine.yaml",
			zoneMapping: false,
		},
		"Zone mapping enabled": {
			fileName:    "test/regions-supporting-machine-with-zones.yaml",
			zoneMapping: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile(tc.fileName, tc.zoneMapping)
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
		})
	}
}
