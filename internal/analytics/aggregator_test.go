package analytics

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
)

const testRegion = "eu-central-1"

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
	region := testRegion
	params := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
	}
	dists := BuildDistributions(params)
	found := false
	for _, d := range dists {
		if d.Parameter == "region" {
			assert.Equal(t, 2, d.Values[testRegion])
			found = true
		}
	}
	assert.True(t, found, "region should appear in distributions")
}

func strPtr(s string) *string { return &s }

// combinedCountFor returns the SetCount for param from AggregateCombined.
func combinedCountFor(prov []ProvisioningParamsWithID, upd []UpdateParamsWithID, param string) int {
	return AggregateCombined(prov, upd).CountFor(param)
}

// provCountFor returns the SetCount for param from AggregateProvisioning.
func provCountFor(prov []ProvisioningParamsWithID, param string) int {
	return AggregateProvisioning(prov).CountFor(param)
}

// distinctUpdateInstancesWithParam counts distinct instance IDs in upd that have param set.
func distinctUpdateInstancesWithParam(upd []UpdateParamsWithID, param string) int {
	seen := make(map[string]struct{})
	for _, u := range upd {
		counts := make(map[string]map[string]int)
		walkFields(u.Params, updatingFieldConfig, counts)
		if _, ok := counts[param]; ok {
			seen[u.InstanceID] = struct{}{}
		}
	}
	return len(seen)
}

// TestAggregateCombined_SetOnlyParam_EqualsProvisioning confirms that for parameters
// that can only be set at provisioning time (not tracked in updatingFieldConfig),
// combined equals provisioning regardless of what updates are present.
func TestAggregateCombined_SetOnlyParam_EqualsProvisioning(t *testing.T) {
	region := testRegion
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i3", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	// Updates on all three instances — but "region" is not in updatingFieldConfig, so updates cannot affect it.
	upd := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{}},
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{}},
		{InstanceID: "i3", Params: internal.UpdatingParametersDTO{}},
	}
	pCount := provCountFor(prov, "region")
	cCount := combinedCountFor(prov, upd, "region")
	assert.Equal(t, pCount, cCount, "set-only param: combined must equal provisioning")
}

// TestAggregateCombined_UpdatableParam_SameInstances checks that when updates set a param
// on the same instances that also provisioned it, combined equals provisioning (no new instances added).
func TestAggregateCombined_UpdatableParam_SameInstances(t *testing.T) {
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}}},
		{InstanceID: "i3", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	// Updates on i1 and i2 only — same instances that already have gvisor from provisioning.
	upd := []UpdateParamsWithID{
		{InstanceID: "i1", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
	}
	pCount := provCountFor(prov, "gvisor")
	cCount := combinedCountFor(prov, upd, "gvisor")
	assert.Equal(t, 2, pCount)
	assert.Equal(t, pCount, cCount, "updatable param on same instances: combined must equal provisioning")
}

// TestAggregateCombined_UpdatableParam_DisjointInstances checks that when updates set a param
// on instances that did NOT have it at provisioning, combined grows beyond provisioning.
// Bound: max(provCount, updateDistinctCount) <= combined <= provCount + updateDistinctCount.
func TestAggregateCombined_UpdatableParam_DisjointInstances(t *testing.T) {
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
		{InstanceID: "i3", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	// Updates on i2 and i3 — different instances from i1 which provisioned with gvisor.
	upd := []UpdateParamsWithID{
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
		{InstanceID: "i3", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
	}
	pCount := provCountFor(prov, "gvisor")
	uCount := distinctUpdateInstancesWithParam(upd, "gvisor")
	cCount := combinedCountFor(prov, upd, "gvisor")

	lower := pCount
	if uCount > lower {
		lower = uCount
	}
	upper := pCount + uCount

	assert.Equal(t, 1, pCount)
	assert.Equal(t, 2, uCount)
	assert.GreaterOrEqual(t, cCount, lower, "combined >= max(provisioning, update distinct instances)")
	assert.LessOrEqual(t, cCount, upper, "combined <= provisioning + update distinct instances")
	assert.Equal(t, 3, cCount, "all three distinct instances have gvisor set")
}

// TestAggregateCombined_UpdatableParam_PartialOverlap confirms the bounds hold when
// updates partially overlap with provisioned instances.
func TestAggregateCombined_UpdatableParam_PartialOverlap(t *testing.T) {
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}}},
		{InstanceID: "i3", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
		{InstanceID: "i4", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{}}},
	}
	// i2 overlaps (already had gvisor); i3 is new via update.
	upd := []UpdateParamsWithID{
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
		{InstanceID: "i3", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
	}
	pCount := provCountFor(prov, "gvisor")
	uCount := distinctUpdateInstancesWithParam(upd, "gvisor")
	cCount := combinedCountFor(prov, upd, "gvisor")

	lower := pCount
	if uCount > lower {
		lower = uCount
	}
	upper := pCount + uCount

	assert.Equal(t, 2, pCount)
	assert.Equal(t, 2, uCount)
	assert.GreaterOrEqual(t, cCount, lower)
	assert.LessOrEqual(t, cCount, upper)
	assert.Equal(t, 3, cCount, "i1+i2 from provisioning, i3 added by update (i2 overlap not double-counted)")
}

// TestBuildDistributions_IncludesCountBehaviorFields confirms that fields with behaviorCount
// (e.g. additionalWorkerNodePools) now appear in distributions.
func TestBuildDistributions_IncludesCountBehaviorFields(t *testing.T) {
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{
			AdditionalWorkerNodePools: []pkg.AdditionalWorkerNodePool{{}, {}},
		}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{
			AdditionalWorkerNodePools: []pkg.AdditionalWorkerNodePool{{}},
		}}},
	}
	dists := BuildDistributions(prov)
	found := false
	for _, d := range dists {
		if d.Parameter == "additionalWorkerNodePools" {
			assert.Equal(t, 1, d.Values["2"], "i1 has 2 pools")
			assert.Equal(t, 1, d.Values["1"], "i2 has 1 pool")
			found = true
		}
	}
	assert.True(t, found, "additionalWorkerNodePools should appear in distributions")
}

// TestBuildDistributions_ParamSetConsistency confirms that every parameter appearing
// in provisioning stats also appears in distributions. BuildDistributions is built
// from provisioning params only, so this is the only guaranteed invariant.
func TestBuildDistributions_ParamSetConsistency(t *testing.T) {
	region := testRegion
	machineType := "m6i.xlarge"
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{
			Region:                    &region,
			MachineType:               &machineType,
			Gvisor:                    &pkg.GvisorDTO{Enabled: true},
			AdditionalWorkerNodePools: []pkg.AdditionalWorkerNodePool{{}},
		}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{
			Region: &region,
		}}},
	}

	provStats := AggregateProvisioning(prov)
	dists := BuildDistributions(prov)

	distParams := make(map[string]struct{})
	for _, d := range dists {
		distParams[d.Parameter] = struct{}{}
	}

	for _, p := range provStats.Parameters {
		_, ok := distParams[p.Parameter]
		assert.True(t, ok, "provisioning param %q must appear in distributions", p.Parameter)
	}
}

// TestBuildDistributions_UpdateOnlyParamAbsentFromDistributions documents that a
// parameter set only via an update operation (never at provisioning time) does not
// appear in distributions, because BuildDistributions takes provisioning params only.
// The UI handles this via the "Include not provided/null" checkbox, which adds
// such combined params to the distribution dropdown showing 100% not-provided.
func TestBuildDistributions_UpdateOnlyParamAbsentFromDistributions(t *testing.T) {
	region := testRegion
	prov := []ProvisioningParamsWithID{
		{InstanceID: "i1", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
		{InstanceID: "i2", Params: internal.ProvisioningParameters{Parameters: pkg.ProvisioningParametersDTO{Region: &region}}},
	}
	upd := []UpdateParamsWithID{
		{InstanceID: "i2", Params: internal.UpdatingParametersDTO{Gvisor: &pkg.GvisorDTO{Enabled: true}}},
	}

	combined := AggregateCombined(prov, upd)
	dists := BuildDistributions(prov)

	distParams := make(map[string]struct{})
	for _, d := range dists {
		distParams[d.Parameter] = struct{}{}
	}

	gvisorInCombined := false
	for _, p := range combined.Parameters {
		if p.Parameter == "gvisor" {
			gvisorInCombined = true
			break
		}
	}
	assert.True(t, gvisorInCombined, "gvisor must appear in combined (set via update on i2)")
	_, inDist := distParams["gvisor"]
	assert.False(t, inDist, "gvisor must not appear in distributions (no instance provisioned with it)")
}
