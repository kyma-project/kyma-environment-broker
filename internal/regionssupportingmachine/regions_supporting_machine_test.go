package regionssupportingmachine

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

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
			result := regionsSupportingMachine.IsSupported(tt.region, tt.machineType)

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
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name        string
		planID      string
		machineType string
		region      string
		expected    []string
	}{
		{
			name:        "AWS plan - region with 3 zones",
			planID:      broker.AWSPlanID,
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"southamerica-east1a", "southamerica-east1b", "southamerica-east1c"},
		},
		{
			name:        "AWS plan - region with 2 zones",
			planID:      broker.AWSPlanID,
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"brazilsoutha", "brazilsouthb"},
		},
		{
			name:        "AWS plan - region with 1 zones",
			planID:      broker.AWSPlanID,
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"uksoutha"},
		},
		{
			name:        "Azure plan - region with 3 zones",
			planID:      broker.AzurePlanID,
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "Azure plan - region with 2 zones",
			planID:      broker.AzurePlanID,
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"a", "b"},
		},
		{
			name:        "Azure plan - region with 1 zones",
			planID:      broker.AzurePlanID,
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"a"},
		},
		{
			name:        "GCP plan - region with 3 zones",
			planID:      broker.GCPPlanID,
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"southamerica-east1-a", "southamerica-east1-b", "southamerica-east1-c"},
		},
		{
			name:        "GCP plan - region with 2 zones",
			planID:      broker.GCPPlanID,
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"brazilsouth-a", "brazilsouth-b"},
		},
		{
			name:        "GCP plan - region with 1 zones",
			planID:      broker.GCPPlanID,
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"uksouth-a"},
		},
		{
			name:        "region with empty list of zones",
			planID:      broker.AzurePlanID,
			machineType: "m8g.large",
			region:      "ap-southeast-1",
			expected:    []string{},
		},
		{
			name:        "region with nil zones",
			planID:      broker.AzurePlanID,
			machineType: "c2d-highmem-32",
			region:      "us-central1",
			expected:    []string{},
		},
		{
			name:        "not supported region",
			planID:      broker.AzurePlanID,
			machineType: "Standard_L8s_v3",
			region:      "westus2",
			expected:    []string{},
		},
		{
			name:        "not supported machine type",
			planID:      broker.AzurePlanID,
			machineType: "notSupportedMachineType",
			region:      "notSupportedRegion",
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result, err := regionsSupportingMachine.AvailableZones(tt.machineType, tt.region, tt.planID)
			assert.NoError(t, err)

			// then
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestAvailableZonesForRegionWith4Zones(t *testing.T) {
	// given
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name     string
		planID   string
		expected []string
	}{
		{
			name:     "AWS plan",
			planID:   broker.AWSPlanID,
			expected: []string{"ap-northeast-1a", "ap-northeast-1b", "ap-northeast-1c", "ap-northeast-1d"},
		},
		{
			name:     "Azure plan",
			planID:   broker.AzurePlanID,
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "GCP plan",
			planID:   broker.GCPPlanID,
			expected: []string{"ap-northeast-1-a", "ap-northeast-1-b", "ap-northeast-1-c", "ap-northeast-1-d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//when
			result, err := regionsSupportingMachine.AvailableZones("m8g", "ap-northeast-1", tt.planID)
			assert.NoError(t, err)

			//then
			assert.Len(t, result, 3)
			for _, v := range result {
				assert.Contains(t, tt.expected, v)
			}
		})
	}
}

func TestAvailableZonesForAzureLite(t *testing.T) {
	// given
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	tests := []struct {
		name        string
		machineType string
		region      string
		expected    []string
	}{
		{
			name:        "region with 4 zones",
			machineType: "m8g",
			region:      "ap-northeast-1",
			expected:    []string{"a", "b", "c", "d"},
		},
		{
			name:        "region with 3 zones",
			machineType: "c2d-highmem-32",
			region:      "southamerica-east1",
			expected:    []string{"a", "b", "c"},
		},
		{
			name:        "region with 2 zones",
			machineType: "Standard_L8s_v3",
			region:      "brazilsouth",
			expected:    []string{"a", "b"},
		},
		{
			name:        "region with 1 zones",
			machineType: "Standard_L8s_v3",
			region:      "uksouth",
			expected:    []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result, err := regionsSupportingMachine.AvailableZones(tt.machineType, tt.region, broker.AzureLitePlanID)
			assert.NoError(t, err)

			//then
			assert.Len(t, result, 1)
			assert.Contains(t, tt.expected, result[0])
		})
	}
}

func TestAvailableZonesForNotSupportedPlan(t *testing.T) {
	// given
	regionsSupportingMachine, err := ReadRegionsSupportingMachineFromFile("test/regions-supporting-machine.yaml")
	require.NoError(t, err)

	_, err = regionsSupportingMachine.AvailableZones("Standard_L8s_v3", "uksouth", "not-supported-plan")
	assert.EqualError(t, err, "plan not-supported-plan not supported")
}
