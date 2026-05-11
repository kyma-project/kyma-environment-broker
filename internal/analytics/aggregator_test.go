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

func TestWalkFields_ModulesDefault(t *testing.T) {
	defaultTrue := true
	dto := pkg.ProvisioningParametersDTO{
		Modules: &pkg.ModulesDTO{Default: &defaultTrue},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["modules"]["default"])
}

func TestWalkFields_ModulesCustom(t *testing.T) {
	defaultFalse := false
	dto := pkg.ProvisioningParametersDTO{
		Modules: &pkg.ModulesDTO{Default: &defaultFalse},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["modules"]["custom"])
}

func TestWalkFields_ModulesNilDefault(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		Modules: &pkg.ModulesDTO{},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["modules"]["custom"])
}

func TestWalkFields_GvisorEnabled(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		Gvisor: &pkg.GvisorDTO{Enabled: true},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["gvisor"]["true"])
}

func TestWalkFields_GvisorDisabled(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		Gvisor: &pkg.GvisorDTO{Enabled: false},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["gvisor"]["false"])
}

func TestWalkFields_ACLWithCIDRs(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		AccessControlList: &pkg.AclDTO{AllowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"}},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["accessControlList"]["2"])
}

func TestWalkFields_ACLEmpty(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		AccessControlList: &pkg.AclDTO{},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["accessControlList"]["0"])
}

func TestWalkFields_NetworkingNodesOnly(t *testing.T) {
	dto := pkg.ProvisioningParametersDTO{
		Networking: &pkg.NetworkingDTO{NodesCidr: "10.250.0.0/22"},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["networking"]["nodes"])
}

func TestWalkFields_NetworkingWithPodsAndServices(t *testing.T) {
	pods := "100.64.0.0/11"
	services := "100.104.0.0/13"
	dto := pkg.ProvisioningParametersDTO{
		Networking: &pkg.NetworkingDTO{
			NodesCidr:    "10.250.0.0/22",
			PodsCidr:     &pods,
			ServicesCidr: &services,
		},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["networking"]["nodes+pods+services"])
}

func TestWalkFields_NetworkingWithDualStack(t *testing.T) {
	dualStack := true
	dto := pkg.ProvisioningParametersDTO{
		Networking: &pkg.NetworkingDTO{
			NodesCidr: "10.250.0.0/22",
			DualStack: &dualStack,
		},
	}
	counts := make(map[string]map[string]int)
	walkFields(dto, provisioningFieldConfig, counts)
	assert.Equal(t, 1, counts["networking"]["nodes+dualStack:true"])
}

func TestAggregateProvisioning_RanksParameters(t *testing.T) {
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}}},
		{InstanceID: "i3", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
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
	params := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m6i.xlarge")}},
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m5.xlarge")}},
		{InstanceID: "i3", Params: internal.UpdatingParametersDTO{}},
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
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
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
