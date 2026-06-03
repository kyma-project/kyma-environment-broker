package analytics

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type fieldBehavior int

const (
	behaviorValue      fieldBehavior = iota // emit field value as string (default)
	behaviorSkip                            // ignore field entirely
	behaviorCount                           // emit slice/array length as value
	behaviorModules                         // emit "default" or "custom"
	behaviorGvisor                          // emit "true" or "false"
	behaviorACL                             // emit CIDR count as string
	behaviorNetworking                      // emit set CIDR fields as "+" joined string with dualStack suffix
	behaviorOIDC                            // emit 0 (nil), 1 (single), or len(List) (multi)
)

// provisioningFieldConfig controls per-field behavior for ProvisioningParametersDTO.
// Fields not listed default to behaviorValue. Keys are JSON tag names.
var provisioningFieldConfig = map[string]fieldBehavior{
	"zones":                     behaviorSkip,
	"targetSecret":              behaviorSkip,
	"kubeconfig":                behaviorSkip,
	"shootName":                 behaviorSkip,
	"shootDomain":               behaviorSkip,
	"administrators":            behaviorCount,
	"additionalWorkerNodePools": behaviorCount,
	"modules":                   behaviorModules,
	"gvisor":                    behaviorGvisor,
	"accessControlList":         behaviorACL,
	"networking":                behaviorNetworking,
	"oidc":                      behaviorOIDC,
}

// updatingFieldConfig controls per-field behavior for UpdatingParametersDTO.
// Only fields that exist on UpdatingParametersDTO and whose absence in an update op
// genuinely means the parameter was nullified should appear here.
var updatingFieldConfig = map[string]fieldBehavior{
	"administrators":            behaviorCount,
	"additionalWorkerNodePools": behaviorCount,
	"gvisor":                    behaviorGvisor,
	"accessControlList":         behaviorACL,
}

// handleStructBehavior handles typed struct behaviors (modules, gvisor, acl, networking)
// by type-asserting the reflect.Value directly. Returns true if the field was handled.
func handleStructBehavior(behavior fieldBehavior, fv reflect.Value, key string, counts map[string]map[string]int) bool {
	if fv.Kind() != reflect.Ptr || fv.IsNil() {
		return behavior == behaviorModules || behavior == behaviorGvisor ||
			behavior == behaviorACL || behavior == behaviorNetworking || behavior == behaviorOIDC
	}
	switch behavior {
	case behaviorModules:
		if m, ok := fv.Interface().(*pkg.ModulesDTO); ok {
			record(counts, key, modulesValue(m))
		}
		return true
	case behaviorGvisor:
		if g, ok := fv.Interface().(*pkg.GvisorDTO); ok {
			record(counts, key, fmt.Sprintf("%t", g.Enabled))
		}
		return true
	case behaviorACL:
		if a, ok := fv.Interface().(*pkg.AclDTO); ok {
			record(counts, key, fmt.Sprintf("%d", len(a.AllowedCIDRs)))
		}
		return true
	case behaviorNetworking:
		if n, ok := fv.Interface().(*pkg.NetworkingDTO); ok {
			if v := networkingValue(n); v != "" {
				record(counts, key, v)
			}
		}
		return true
	case behaviorOIDC:
		if o, ok := fv.Interface().(*pkg.OIDCConnectDTO); ok {
			record(counts, key, fmt.Sprintf("%d", oidcCount(o)))
		}
		return true
	}
	return false
}

// walkFields reflects over a struct, applies fieldConfig, and populates counts:
//
//	counts[jsonName][value] = occurrenceCount
//
// The value emitted depends on the field's configured behavior: scalars are
// stringified as-is (behaviorValue), slices/arrays emit their length, and typed
// struct behaviors (behaviorModules, behaviorGvisor, behaviorACL, behaviorNetworking,
// behaviorOIDC) apply domain-specific transformations before recording. Fields
// listed as behaviorSkip are silently ignored. Nil pointers and zero-value
// non-pointer fields produce no entry.
func walkFields(v interface{}, config map[string]fieldBehavior, counts map[string]map[string]int) {
	rv := reflect.ValueOf(v)
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)

		// Recurse into anonymous (embedded) structs only
		if field.Anonymous {
			walkFields(fv.Interface(), config, counts)
			continue
		}

		// Derive key from JSON tag, falling back to field name
		jsonName := field.Name
		if tag, ok := field.Tag.Lookup("json"); ok {
			if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
				jsonName = name
			}
		}

		behavior, ok := config[jsonName]
		if !ok {
			behavior = behaviorValue
		}
		if behavior == behaviorSkip {
			continue
		}

		// Handle typed behaviors that work directly on the pointer before dereferencing
		if handled := handleStructBehavior(behavior, fv, jsonName, counts); handled {
			continue
		}

		// Dereference pointers; skip nil
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		} else {
			// Skip zero/empty values for non-pointer fields
			if fv.IsZero() {
				continue
			}
		}

		var value string
		switch {
		case fv.Kind() == reflect.Slice || fv.Kind() == reflect.Array:
			value = fmt.Sprintf("%d", fv.Len())
		case fv.Kind() == reflect.Struct:
			value = "1"
		case behavior == behaviorCount:
			// behaviorCount on a non-slice/non-struct: skip
			continue
		default: // behaviorValue on scalar
			switch fv.Kind() {
			case reflect.String:
				value = fv.String()
				if value == "" {
					continue
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				value = fmt.Sprintf("%d", fv.Int())
			case reflect.Bool:
				value = fmt.Sprintf("%t", fv.Bool())
			default:
				continue
			}
		}

		record(counts, jsonName, value)
	}
}

// modulesValue returns "default" if modules.Default is true, otherwise "custom".
func modulesValue(m *pkg.ModulesDTO) string {
	if m.Default != nil && *m.Default {
		return "default"
	}
	return "custom"
}

// oidcCount returns the number of OIDC providers configured:
// 0 if nil, 1 for the legacy single-provider form, len(List) for the multi-provider form.
func oidcCount(o *pkg.OIDCConnectDTO) int {
	if o == nil {
		return 0
	}
	if len(o.List) > 0 {
		return len(o.List)
	}
	if o.OIDCConfigDTO != nil {
		return 1
	}
	return 0
}

// networkingValue returns a "+" joined string of the CIDR fields that are set,
// with "dualStack:<bool>" appended if dualStack is explicitly configured.
// nodes is mandatory in the schema but may be empty in tests; if nothing is set, returns "".
func networkingValue(n *pkg.NetworkingDTO) string {
	var parts []string
	if n.NodesCidr != "" {
		parts = append(parts, "nodes")
	}
	if n.PodsCidr != nil {
		parts = append(parts, "pods")
	}
	if n.ServicesCidr != nil {
		parts = append(parts, "services")
	}
	if n.DualStack != nil {
		parts = append(parts, fmt.Sprintf("dualStack:%t", *n.DualStack))
	}
	return strings.Join(parts, "+")
}

// record adds value to counts[key], initialising the inner map if needed.
func record(counts map[string]map[string]int, key, value string) {
	if _, ok := counts[key]; !ok {
		counts[key] = make(map[string]int)
	}
	counts[key][value]++
}

// buildCounts walks all params once and returns field-value occurrence counts.
func buildCounts(params []ProvisioningParamsWithID) map[string]map[string]int {
	counts := make(map[string]map[string]int)
	for _, p := range params {
		walkFields(p.Params.Parameters, provisioningFieldConfig, counts)
	}
	return counts
}

// AggregateProvisioning computes parameter usage stats from a slice of ProvisioningParameters.
func AggregateProvisioning(params []ProvisioningParamsWithID) ParameterStats {
	plain := PlainProvisioningParams(params)
	counts := make(map[string]map[string]int)
	for _, p := range plain {
		walkFields(p.Parameters, provisioningFieldConfig, counts)
	}
	return toParameterStats(counts, len(plain))
}

// AggregateUpdates computes parameter usage stats from a slice of UpdateParamsWithID.
func AggregateUpdates(params []UpdateParamsWithID) ParameterStats {
	counts := make(map[string]map[string]int)
	for _, p := range params {
		walkFields(p.Params, updatingFieldConfig, counts)
	}
	return toParameterStats(counts, len(params))
}

// AggregateCombined computes per-instance parameter usage: an instance is counted as
// "using" a parameter if it was set in its provisioning operation OR in any update operation.
// total = number of unique active instances (from provParams).
func AggregateCombined(provParams []ProvisioningParamsWithID, updateParams []UpdateParamsWithID) ParameterStats {
	// instancesWithParam[param] = set of instance IDs that have the param set
	instancesWithParam := make(map[string]map[string]struct{})

	record := func(instanceID, param string) {
		if _, ok := instancesWithParam[param]; !ok {
			instancesWithParam[param] = make(map[string]struct{})
		}
		instancesWithParam[param][instanceID] = struct{}{}
	}

	for _, p := range provParams {
		counts := make(map[string]map[string]int)
		walkFields(p.Params.Parameters, provisioningFieldConfig, counts)
		for param := range counts {
			record(p.InstanceID, param)
		}
	}

	for _, p := range updateParams {
		counts := make(map[string]map[string]int)
		walkFields(p.Params, updatingFieldConfig, counts)
		for param := range counts {
			record(p.InstanceID, param)
		}
	}

	total := len(provParams)
	var result []ParameterStat
	for param, instances := range instancesWithParam {
		result = append(result, ParameterStat{
			Parameter: param,
			SetCount:  len(instances),
			Total:     total,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].SetCount != result[j].SetCount {
			return result[i].SetCount > result[j].SetCount
		}
		return result[i].Parameter < result[j].Parameter
	})
	return ParameterStats{Parameters: result}
}

// toParameterStats converts raw counts into a ranked ParameterStats list.
func toParameterStats(counts map[string]map[string]int, total int) ParameterStats {
	var result []ParameterStat
	for param, values := range counts {
		setCount := 0
		for _, c := range values {
			setCount += c
		}
		result = append(result, ParameterStat{
			Parameter: param,
			SetCount:  setCount,
			Total:     total,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].SetCount != result[j].SetCount {
			return result[i].SetCount > result[j].SetCount
		}
		return result[i].Parameter < result[j].Parameter
	})
	return ParameterStats{Parameters: result}
}

// BuildDistributions computes value breakdowns for all tracked fields from provisioning params.
// All non-skip behaviors are included; behaviorCount fields emit their numeric length as the bucket value.
func BuildDistributions(params []ProvisioningParamsWithID) []DistributionStat {
	counts := buildCounts(params)
	fields := make([]string, 0, len(counts))
	for field := range counts {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	result := make([]DistributionStat, 0, len(fields))
	for _, field := range fields {
		result = append(result, DistributionStat{
			Parameter: field,
			Values:    counts[field],
		})
	}
	return result
}

// BuildPlanRegionIndex builds a map of plan name → sorted distinct regions,
// plus a sorted list of all plan names. planIDToName maps plan UUID → plan name.
// The special key "" in the returned map contains all regions across all plans.
func BuildPlanRegionIndex(params []ProvisioningParamsWithID, planIDToName map[string]string) ([]string, map[string][]string) {
	planSet := make(map[string]struct{})
	// plan → region → present
	byPlan := make(map[string]map[string]struct{})

	for _, p := range params {
		name := planIDToName[p.Params.PlanID]
		if name == "" {
			name = p.Params.PlanID // fallback to raw ID
		}
		planSet[name] = struct{}{}

		region := ""
		if p.Params.Parameters.Region != nil {
			region = *p.Params.Parameters.Region
		}

		if _, ok := byPlan[name]; !ok {
			byPlan[name] = make(map[string]struct{})
		}
		if region != "" {
			byPlan[name][region] = struct{}{}
		}
	}

	// sorted plan list
	plans := make([]string, 0, len(planSet))
	for name := range planSet {
		plans = append(plans, name)
	}
	sort.Strings(plans)

	// build regionsByPlan with sorted slices; also collect all regions
	allRegions := make(map[string]struct{})
	regionsByPlan := make(map[string][]string, len(byPlan)+1)
	for plan, regions := range byPlan {
		sorted := make([]string, 0, len(regions))
		for r := range regions {
			sorted = append(sorted, r)
			allRegions[r] = struct{}{}
		}
		sort.Strings(sorted)
		regionsByPlan[plan] = sorted
	}

	// "" key = all regions
	all := make([]string, 0, len(allRegions))
	for r := range allRegions {
		all = append(all, r)
	}
	sort.Strings(all)
	regionsByPlan[""] = all

	return plans, regionsByPlan
}

// FilterByPlan returns only params matching the given plan name.
// planIDToName maps plan UUID → plan name.
func FilterByPlan(params []ProvisioningParamsWithID, plan string, planIDToName map[string]string) []ProvisioningParamsWithID {
	result := params[:0:0]
	for _, p := range params {
		name := planIDToName[p.Params.PlanID]
		if name == "" {
			name = p.Params.PlanID
		}
		if name == plan {
			result = append(result, p)
		}
	}
	return result
}

// FilterByRegion returns only params where Parameters.Region matches the given region.
func FilterByRegion(params []ProvisioningParamsWithID, region string) []ProvisioningParamsWithID {
	result := params[:0:0]
	for _, p := range params {
		if p.Params.Parameters.Region != nil && *p.Params.Parameters.Region == region {
			result = append(result, p)
		}
	}
	return result
}

// isParamSet returns true if the given parameter name has any recorded value when
// walking the struct with the given field config. Uses the same walkFields logic as
// all other aggregation, so null/zero values are treated consistently.
func isParamSet(v interface{}, config map[string]fieldBehavior, param string) bool {
	counts := make(map[string]map[string]int)
	walkFields(v, config, counts)
	_, ok := counts[param]
	return ok
}

// BuildTrend computes the daily cumulative count of active instances that have the
// given parameter set, derived from the ordered sequence of op events.
// Events must be sorted by created_at ASC (as returned by FetchOpEventsInRange).
// For each instance the function tracks whether the parameter is currently set,
// emits +1 / -1 / 0 deltas, buckets them by day, and returns the running total.
// TrendPoint.Total holds the cumulative count of instances provisioned up to that day
// (never decremented when an instance is deprovisioned).
func BuildTrend(events []OpEvent, param string) TrendStat {
	// per-instance: is the param currently set?
	instanceState := make(map[string]bool)

	// day → net delta for that day
	dayDelta := make(map[string]int)
	// day → new instances provisioned on that day
	dayProvisioned := make(map[string]int)

	for _, ev := range events {
		wasSet := instanceState[ev.InstanceID]
		var nowSet bool

		switch ev.Type {
		case "provision":
			p, err := parseProvisioningParameters(ev.RawParams)
			if err != nil {
				continue
			}
			nowSet = isParamSet(p.Parameters, provisioningFieldConfig, param)
			dayProvisioned[ev.CreatedAt]++
		case "update":
			var op internal.Operation
			if err := json.Unmarshal([]byte(ev.RawParams), &op); err != nil {
				continue
			}
			if isParamSet(op.UpdatingParameters, updatingFieldConfig, param) {
				nowSet = true
			} else if _, inConfig := updatingFieldConfig[param]; inConfig {
				// param is updatable and absent from this op → nullified
				nowSet = false
			} else {
				// param not updatable — state unchanged
				nowSet = wasSet
			}
		default:
			continue
		}

		instanceState[ev.InstanceID] = nowSet

		delta := 0
		if nowSet && !wasSet {
			delta = 1
		} else if !nowSet && wasSet {
			delta = -1
		}
		if delta != 0 {
			dayDelta[ev.CreatedAt] += delta
		}
	}

	// Collect the first and last day across all events to build a continuous daily range.
	allEventDays := make(map[string]struct{})
	for d := range dayDelta {
		allEventDays[d] = struct{}{}
	}
	for d := range dayProvisioned {
		allEventDays[d] = struct{}{}
	}
	if len(allEventDays) == 0 {
		return TrendStat{Parameter: param, Points: nil}
	}
	eventDaysSorted := make([]string, 0, len(allEventDays))
	for d := range allEventDays {
		eventDaysSorted = append(eventDaysSorted, d)
	}
	sort.Strings(eventDaysSorted)

	// Build a continuous day-by-day sequence from first to last event day so the
	// chart X-axis is a linear time scale (gaps appear as flat segments, not compressed).
	firstDay, lastDay := eventDaysSorted[0], eventDaysSorted[len(eventDaysSorted)-1]
	allDays := expandDayRange(firstDay, lastDay)

	points := make([]TrendPoint, 0, len(allDays))
	running := 0
	runningTotal := 0
	for _, day := range allDays {
		running += dayDelta[day]
		runningTotal += dayProvisioned[day]
		points = append(points, TrendPoint{Date: day, Count: running, Total: runningTotal})
	}

	return TrendStat{Parameter: param, Points: points}
}

// BuildTrends computes daily trend lines for each of the given parameters.
func BuildTrends(events []OpEvent, params []string) []TrendStat {
	result := make([]TrendStat, 0, len(params))
	for _, p := range params {
		result = append(result, BuildTrend(events, p))
	}
	return result
}

// TrendParamsFrom returns the sorted list of parameter names from combined stats.
// Used to compute trends for all parameters that actually appear in the data.
func TrendParamsFrom(combined ParameterStats) []string {
	params := make([]string, 0, len(combined.Parameters))
	for _, p := range combined.Parameters {
		params = append(params, p.Parameter)
	}
	return params
}

// expandDayRange returns every calendar day from first to last inclusive,
// in YYYY-MM-DD format. Both inputs must be valid YYYY-MM-DD strings.
func expandDayRange(first, last string) []string {
	const layout = "2006-01-02"
	t, err := time.Parse(layout, first)
	if err != nil {
		return []string{first}
	}
	end, err := time.Parse(layout, last)
	if err != nil {
		return []string{first}
	}
	var days []string
	for !t.After(end) {
		days = append(days, t.Format(layout))
		t = t.AddDate(0, 0, 1)
	}
	return days
}
