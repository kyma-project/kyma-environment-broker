package analytics

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
)

func TestWalkFields_SkipsConfiguredFields(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		Zones: []string{"eu-central-1a"},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	_, found := counts["zones"]
	assert.False(t, found, "zones should be skipped")
}

func TestWalkFields_CountsArrayLength(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		RuntimeAdministrators: []string{"admin1", "admin2"},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["administrators"]["2"])
}

func TestWalkFields_EmitsStringValue(t *testing.T) {
	machineType := "m6i.xlarge"
	dto := pkg.ProvisioningParametersDTO{
		MachineType: &machineType,
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["machineType"]["m6i.xlarge"])
}

func TestWalkFields_SkipsNilPointers(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	_, found := counts["machineType"]
	assert.False(t, found, "nil pointer fields should not be recorded")
}

func TestWalkFields_EmitsNumericValue(t *testing.T) {
	min := 3
	dto := pkg.ProvisioningParametersDTO{
		AutoScalerParameters: pkg.AutoScalerParameters{AutoScalerMin: &min},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["autoScalerMin"]["3"])
}

func TestAggregateProvisioning_RanksParameters(t *testing.T) {
	params := []internal.ProvisioningParameters{
		{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}},
		{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}},
		{Parameters: pkg.ProvisioningParametersDTO{}},
	}
	stats := AggregateProvisioning(params)
	assert.Equal(t, 3, stats.Parameters[0].Total)
	found := false
	for _, p := range stats.Parameters {
		if p.Parameter == "machineType" {
			assert.Equal(t, 2, p.SetCount)
			found = true
		}
	}
	assert.True(t, found)
}

func TestAggregateUpdates_CountsSetFields(t *testing.T) {
	params := []internal.UpdatingParametersDTO{
		{MachineType: strPtr("m6i.xlarge")},
		{MachineType: strPtr("m5.xlarge")},
		{},
	}
	stats := AggregateUpdates(params)
	assert.Equal(t, 3, stats.Parameters[0].Total)
	found := false
	for _, p := range stats.Parameters {
		if p.Parameter == "machineType" {
			assert.Equal(t, 2, p.SetCount)
			found = true
		}
	}
	assert.True(t, found)
}

func TestBuildDistributions_IncludesRegion(t *testing.T) {
	region := "eu-central-1"
	params := []internal.ProvisioningParameters{
		{Parameters: pkg.ProvisioningParametersDTO{Region: &region}},
		{Parameters: pkg.ProvisioningParametersDTO{Region: &region}},
	}
	dists := BuildDistributions(params)
	found := false
	for _, d := range dists {
		if d.Parameter == "region" {
			assert.Equal(t, 2, d.Values["eu-central-1"])
			found = true
		}
	}
	assert.True(t, found, "region should appear in distributions")
}

func strPtr(s string) *string { return &s }
