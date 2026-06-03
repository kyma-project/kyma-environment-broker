package analytics

import (
	"encoding/json"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWalkFields_AggregatesAcrossMultipleInstances(t *testing.T) {
	counts := make(map[string]map[string]int)
	for i := 0; i < 3; i++ {
		min := 3
		dto := pkg.ProvisioningParametersDTO{
			AutoScalerParameters: pkg.AutoScalerParameters{AutoScalerMin: &min},
		}
		walkFields(dto, provisioningFieldConfig, counts)
	}
	assert.Equal(t, 3, counts["autoScalerMin"]["3"])
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
	// every parameter entry must carry the full total
	for _, p := range stats.Parameters {
		assert.Equal(t, 3, p.Total, "parameter %s: expected Total=3", p.Parameter)
	}
	// machineType was set on 2 of 3 → SetCount=2
	assert.Equal(t, 2, stats.CountFor("machineType"))
	// highest-SetCount parameter must be first
	if len(stats.Parameters) > 1 {
		assert.GreaterOrEqual(t, stats.Parameters[0].SetCount, stats.Parameters[1].SetCount)
	}
}

func TestAggregateUpdates_CountsSetFields(t *testing.T) {
	params := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m6i.xlarge")}},
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m5.xlarge")}},
		{InstanceID: "i3", Params: internal.UpdatingParametersDTO{}},
	}
	stats := AggregateUpdates(params)
	// every parameter entry must carry the full total
	for _, p := range stats.Parameters {
		assert.Equal(t, 3, p.Total, "parameter %s: expected Total=3", p.Parameter)
	}
	// machineType was set on 2 of 3 update ops → SetCount=2
	assert.Equal(t, 2, stats.CountFor("machineType"))
	// highest-SetCount parameter must be first
	if len(stats.Parameters) > 1 {
		assert.GreaterOrEqual(t, stats.Parameters[0].SetCount, stats.Parameters[1].SetCount)
	}
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

// ---------------------------------------------------------------------------
// AggregateCombined
// ---------------------------------------------------------------------------

func TestAggregateCombined_ProvisioningOnly(t *testing.T) {
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	stats := AggregateCombined(params, nil)
	assert.Equal(t, 2, stats.Parameters[0].Total)
	assert.Equal(t, 1, stats.CountFor("machineType"))
}

func TestAggregateCombined_UpdateOnly(t *testing.T) {
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	updateParams := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m6i.xlarge")}},
	}
	stats := AggregateCombined(provParams, updateParams)
	assert.Equal(t, 2, stats.Parameters[0].Total)
	assert.Equal(t, 1, stats.CountFor("machineType"))
}

func TestAggregateCombined_UnionAcrossProvAndUpdate(t *testing.T) {
	// i1 has machineType in provisioning, i2 has it in update → SetCount should be 2
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	updateParams := []UpdateParamsWithID{
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m5.xlarge")}},
	}
	stats := AggregateCombined(provParams, updateParams)
	assert.Equal(t, 2, stats.CountFor("machineType"))
}

func TestAggregateCombined_InstanceCountedOnceForSameParam(t *testing.T) {
	// i1 has machineType in both provisioning AND an update → still counts as 1
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}}},
	}
	updateParams := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{MachineType: strPtr("m5.xlarge")}},
	}
	stats := AggregateCombined(provParams, updateParams)
	assert.Equal(t, 1, stats.CountFor("machineType"))
}

func TestAggregateCombined_TotalIsProvisioningCount(t *testing.T) {
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr("m6i.xlarge")}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
		{InstanceID: "i3", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	stats := AggregateCombined(provParams, nil)
	for _, p := range stats.Parameters {
		assert.Equal(t, 3, p.Total, "Total must equal the number of unique provisioned instances")
	}
}

func TestAggregateCombined_SortedBySetCountDescThenName(t *testing.T) {
	machineType := "m6i.xlarge"
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: &machineType}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{MachineType: &machineType}}},
	}
	updateParams := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{
			Gvisor: &pkg.GvisorDTO{Enabled: true},
		}},
	}
	stats := AggregateCombined(provParams, updateParams)
	// machineType SetCount=2, gvisor SetCount=1 → machineType must come first
	if len(stats.Parameters) >= 2 {
		assert.GreaterOrEqual(t, stats.Parameters[0].SetCount, stats.Parameters[1].SetCount)
	}
	assert.Equal(t, "machineType", stats.Parameters[0].Parameter)
}

// TestAggregateCombined_DecreaseableParamNullifiedByUpdate verifies that AggregateCombined
// does NOT track nullification. An instance provisioned WITH gvisor and then updated WITHOUT
// gvisor (which would nullify it in BuildTrend) is still counted in Combined — the union
// only grows, never shrinks.
func TestAggregateCombined_DecreaseableParamNullifiedByUpdate(t *testing.T) {
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{
			Gvisor: &pkg.GvisorDTO{Enabled: true},
		}}},
	}
	// Update without gvisor — gvisor is in updatingFieldConfig so BuildTrend would nullify it,
	// but AggregateCombined simply does a union of "ever set in any op".
	updateParams := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{}},
	}
	stats := AggregateCombined(provParams, updateParams)
	// The provisioning op put i1 in the gvisor set; the update op does not remove it.
	assert.Equal(t, 1, stats.CountFor("gvisor"), "provisioning set gvisor; subsequent nullifying update must not remove instance from Combined")
}

// TestAggregateCombined_DecreaseableParamSetByUpdateThenNullified verifies that an instance
// added to the Combined set via an update op is never removed, even if a later update op
// nullifies the same parameter.
func TestAggregateCombined_DecreaseableParamSetByUpdateThenNullified(t *testing.T) {
	// Provision without gvisor, set it via update, then nullify via a second update.
	provParams := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	updateParams := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}}, // sets gvisor
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{}},                                       // nullifies gvisor
	}
	stats := AggregateCombined(provParams, updateParams)
	// The second update op did not set gvisor, but the first did — i1 remains in the set.
	assert.Equal(t, 1, stats.CountFor("gvisor"), "once added by an update op, instance must remain in Combined even after a nullifying update")
}

// ---------------------------------------------------------------------------
// BuildTrend helpers
// ---------------------------------------------------------------------------

// provEvent creates an OpEvent carrying a provision operation.
func provEvent(instanceID, day, machineType string) OpEvent {
	p := internal.ProvisioningParameters{
		Parameters: pkg.ProvisioningParametersDTO{MachineType: &machineType},
	}
	raw, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "provision", RawParams: string(raw)}
}

// provEventNoParam creates an OpEvent for a provision op without machineType.
func provEventNoParam(instanceID, day string) OpEvent {
	p := internal.ProvisioningParameters{}
	raw, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "provision", RawParams: string(raw)}
}

// updateEvent creates an OpEvent carrying an update operation setting machineType.
func updateEvent(instanceID, day, machineType string) OpEvent {
	op := internal.Operation{
		UpdatingParameters: internal.UpdatingParametersDTO{MachineType: &machineType},
	}
	raw, err := json.Marshal(op)
	if err != nil {
		panic(err)
	}
	return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "update", RawParams: string(raw)}
}

// updateEventNoParam creates an OpEvent for an update op that does NOT set machineType.
func updateEventNoParam(instanceID, day string) OpEvent {
	op := internal.Operation{}
	raw, err := json.Marshal(op)
	if err != nil {
		panic(err)
	}
	return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "update", RawParams: string(raw)}
}

// provEventGvisor creates an OpEvent for a provision op that sets gvisor.
func provEventGvisor(instanceID, day string) OpEvent {
	p := internal.ProvisioningParameters{
		Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}},
	}
	raw, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "provision", RawParams: string(raw)}
}

// updateEventGvisor creates an OpEvent for an update op that sets gvisor.
func updateEventGvisor(instanceID, day string) OpEvent {
	op := internal.Operation{
		UpdatingParameters: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}},
	}
	raw, err := json.Marshal(op)
	if err != nil {
		panic(err)
	}
	return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "update", RawParams: string(raw)}
}

// ---------------------------------------------------------------------------
// BuildTrend tests
// ---------------------------------------------------------------------------

func TestBuildTrend_ParamSetAtProvision(t *testing.T) {
	events := []OpEvent{
		provEvent("i1", "2024-01-01", "m6i.xlarge"),
	}
	trend := BuildTrend(events, "machineType")
	require.Len(t, trend.Points, 1)
	assert.Equal(t, "2024-01-01", trend.Points[0].Date)
	assert.Equal(t, 1, trend.Points[0].Count)
	assert.Equal(t, 1, trend.Points[0].Total)
}

func TestBuildTrend_ParamNotSetAtProvision(t *testing.T) {
	events := []OpEvent{
		provEventNoParam("i1", "2024-01-01"),
	}
	trend := BuildTrend(events, "machineType")
	// one provisioning day but no delta → Count=0, Total=1
	require.Len(t, trend.Points, 1)
	assert.Equal(t, 0, trend.Points[0].Count)
	assert.Equal(t, 1, trend.Points[0].Total)
}

func TestBuildTrend_ParamAddedByUpdate(t *testing.T) {
	// provision without param, then update sets it
	events := []OpEvent{
		provEventNoParam("i1", "2024-01-01"),
		updateEvent("i1", "2024-01-02", "m6i.xlarge"),
	}
	trend := BuildTrend(events, "machineType")
	require.Len(t, trend.Points, 2)
	assert.Equal(t, 0, trend.Points[0].Count) // day 1: no param yet
	assert.Equal(t, 1, trend.Points[1].Count) // day 2: update sets it → +1
}

func TestBuildTrend_ParamRemovedByUpdate(t *testing.T) {
	// provision with param, then update clears it (machineType is NOT in updatingFieldConfig,
	// so its absence in an update op means "state unchanged" — use gvisor which IS updatable)
	provWithGvisor := func(instanceID, day string) OpEvent {
		g := true
		p := internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: g}},
		}
		raw, _ := json.Marshal(p)
		return OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "provision", RawParams: string(raw)}
	}
	events := []OpEvent{
		provWithGvisor("i1", "2024-01-01"),       // gvisor set → Count=1
		updateEventNoParam("i1", "2024-01-02"),    // update without gvisor → nullified (in updatingFieldConfig) → Count=0
	}
	trend := BuildTrend(events, "gvisor")
	require.Len(t, trend.Points, 2)
	assert.Equal(t, 1, trend.Points[0].Count) // day 1: provisioned with gvisor
	assert.Equal(t, 0, trend.Points[1].Count) // day 2: update without gvisor nullifies it
}

func TestBuildTrend_NonUpdatableParamPreservedThroughUpdate(t *testing.T) {
	// machineType is NOT in updatingFieldConfig; an update op without it produces
	// zero delta. Since day 2 has no provision and no delta, no point is emitted for it.
	// The running count after day 1 remains 1.
	events := []OpEvent{
		provEvent("i1", "2024-01-01", "m6i.xlarge"),
		updateEventNoParam("i1", "2024-01-02"), // zero delta — no new TrendPoint
	}
	trend := BuildTrend(events, "machineType")
	// Only one point (day 1); no point for day 2 because delta=0 and no provision occurred.
	require.Len(t, trend.Points, 1)
	assert.Equal(t, 1, trend.Points[0].Count)
}

func TestBuildTrend_TotalAccumulatesProvisions(t *testing.T) {
	events := []OpEvent{
		provEvent("i1", "2024-01-01", "m6i.xlarge"),
		provEvent("i2", "2024-01-02", "m6i.xlarge"),
		provEvent("i3", "2024-01-02", "m6i.xlarge"),
	}
	trend := BuildTrend(events, "machineType")
	require.Len(t, trend.Points, 2)
	assert.Equal(t, 1, trend.Points[0].Total) // 1 provision on day 1
	assert.Equal(t, 3, trend.Points[1].Total) // 2 more on day 2 → cumulative 3
}

func TestBuildTrend_MultipleInstancesMultipleDays(t *testing.T) {
	events := []OpEvent{
		provEvent("i1", "2024-01-01", "m6i.xlarge"), // Count: 1
		provEventNoParam("i2", "2024-01-01"),         // Count: still 1
		provEvent("i3", "2024-01-02", "m6i.xlarge"), // Count: 2
	}
	trend := BuildTrend(events, "machineType")
	require.Len(t, trend.Points, 2)
	assert.Equal(t, 1, trend.Points[0].Count)
	assert.Equal(t, 2, trend.Points[1].Count)
	assert.Equal(t, 2, trend.Points[0].Total) // 2 provisioned on day 1
	assert.Equal(t, 3, trend.Points[1].Total) // +1 on day 2 → cumulative 3
}

// ---------------------------------------------------------------------------
// FilterByPlan
// ---------------------------------------------------------------------------

func TestFilterByPlan_MatchByName(t *testing.T) {
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "plan-uuid-aws"}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{PlanID: "plan-uuid-gcp"}},
	}
	nameMap := map[string]string{"plan-uuid-aws": "aws", "plan-uuid-gcp": "gcp"}
	result := FilterByPlan(params, "aws", nameMap)
	require.Len(t, result, 1)
	assert.Equal(t, "i1", result[0].InstanceID)
}

func TestFilterByPlan_NoMatch(t *testing.T) {
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "plan-uuid-aws"}},
	}
	result := FilterByPlan(params, "gcp", map[string]string{"plan-uuid-aws": "aws"})
	assert.Empty(t, result)
}

func TestFilterByPlan_FallsBackToRawPlanID(t *testing.T) {
	// Plan ID not in nameMap → raw UUID used as name
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "unknown-uuid"}},
	}
	result := FilterByPlan(params, "unknown-uuid", map[string]string{})
	require.Len(t, result, 1)
	assert.Equal(t, "i1", result[0].InstanceID)
}

// ---------------------------------------------------------------------------
// FilterByRegion
// ---------------------------------------------------------------------------

func TestFilterByRegion_MatchByRegion(t *testing.T) {
	region1 := "eu-central-1"
	region2 := "us-east-1"
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region1}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region2}}},
	}
	result := FilterByRegion(params, "eu-central-1")
	require.Len(t, result, 1)
	assert.Equal(t, "i1", result[0].InstanceID)
}

func TestFilterByRegion_NilRegionSkipped(t *testing.T) {
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	result := FilterByRegion(params, "eu-central-1")
	assert.Empty(t, result)
}

func TestFilterByRegion_NoMatch(t *testing.T) {
	region := "us-east-1"
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
	}
	result := FilterByRegion(params, "eu-central-1")
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// BuildPlanRegionIndex
// ---------------------------------------------------------------------------

func TestBuildPlanRegionIndex_SortedPlans(t *testing.T) {
	region := "eu-central-1"
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "uuid-b", Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{PlanID: "uuid-a", Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
	}
	nameMap := map[string]string{"uuid-a": "aws", "uuid-b": "gcp"}
	plans, _ := BuildPlanRegionIndex(params, nameMap)
	assert.Equal(t, []string{"aws", "gcp"}, plans)
}

func TestBuildPlanRegionIndex_SortedRegionsPerPlan(t *testing.T) {
	regionB := "us-east-1"
	regionA := "eu-central-1"
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "uuid-aws", Parameters: pkg.ProvisioningParametersDTO{Region: &regionB}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{PlanID: "uuid-aws", Parameters: pkg.ProvisioningParametersDTO{Region: &regionA}}},
	}
	nameMap := map[string]string{"uuid-aws": "aws"}
	_, byPlan := BuildPlanRegionIndex(params, nameMap)
	assert.Equal(t, []string{"eu-central-1", "us-east-1"}, byPlan["aws"])
}

func TestBuildPlanRegionIndex_AllRegionsKey(t *testing.T) {
	regionA := "eu-central-1"
	regionB := "us-east-1"
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "uuid-aws", Parameters: pkg.ProvisioningParametersDTO{Region: &regionA}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{PlanID: "uuid-gcp", Parameters: pkg.ProvisioningParametersDTO{Region: &regionB}}},
	}
	nameMap := map[string]string{"uuid-aws": "aws", "uuid-gcp": "gcp"}
	_, byPlan := BuildPlanRegionIndex(params, nameMap)
	// "" key must contain all regions across all plans, sorted
	assert.Equal(t, []string{"eu-central-1", "us-east-1"}, byPlan[""])
}

func TestBuildPlanRegionIndex_NilRegionExcludedFromLists(t *testing.T) {
	region := "eu-central-1"
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{PlanID: "uuid-aws", Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{PlanID: "uuid-aws", Parameters: pkg.ProvisioningParametersDTO{}}}, // nil region
	}
	nameMap := map[string]string{"uuid-aws": "aws"}
	plans, byPlan := BuildPlanRegionIndex(params, nameMap)
	// plan still appears (it has instances)
	assert.Equal(t, []string{"aws"}, plans)
	// nil region is excluded from the region list
	assert.Equal(t, []string{"eu-central-1"}, byPlan["aws"])
}
