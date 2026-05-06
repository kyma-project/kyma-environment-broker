package analytics

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

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
}

// updatingFieldConfig controls per-field behavior for UpdatingParametersDTO.
var updatingFieldConfig = map[string]fieldBehavior{
	"administrators":            behaviorCount,
	"additionalWorkerNodePools": behaviorCount,
	"modules":                   behaviorModules,
	"gvisor":                    behaviorGvisor,
	"accessControlList":         behaviorACL,
}

// handleStructBehavior handles typed struct behaviors (modules, gvisor, acl, networking)
// by type-asserting the reflect.Value directly. Returns true if the field was handled.
func handleStructBehavior(behavior fieldBehavior, fv reflect.Value, key string, counts map[string]map[string]int) bool {
	if fv.Kind() != reflect.Ptr || fv.IsNil() {
		return behavior == behaviorModules || behavior == behaviorGvisor ||
			behavior == behaviorACL || behavior == behaviorNetworking
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
	}
	return false
}

// walkFields reflects over a struct, applies fieldConfig, and populates counts:
//
//	counts[jsonName][value] = occurrenceCount
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
func buildCounts(params []internal.ProvisioningParameters) map[string]map[string]int {
	counts := make(map[string]map[string]int)
	for _, p := range params {
		walkFields(p.Parameters, provisioningFieldConfig, counts)
	}
	return counts
}

// AggregateProvisioning computes parameter usage stats from a slice of ProvisioningParameters.
func AggregateProvisioning(params []internal.ProvisioningParameters) ParameterStats {
	return toParameterStats(buildCounts(params), len(params))
}

// AggregateUpdates computes parameter usage stats from a slice of UpdatingParametersDTO.
func AggregateUpdates(params []internal.UpdatingParametersDTO) ParameterStats {
	counts := make(map[string]map[string]int)
	total := len(params)
	for _, p := range params {
		walkFields(p, updatingFieldConfig, counts)
	}
	return toParameterStats(counts, total)
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

// BuildDistributions computes value breakdowns for selected distribution fields.
func BuildDistributions(params []internal.ProvisioningParameters) []DistributionStat {
	distributionFields := []string{"machineType", "region", "autoScalerMin", "autoScalerMax"}
	counts := buildCounts(params)
	var result []DistributionStat
	for _, field := range distributionFields {
		if values, ok := counts[field]; ok {
			result = append(result, DistributionStat{
				Parameter: field,
				Values:    values,
			})
		}
	}
	return result
}

// BuildPlanRegionIndex builds a map of plan name → sorted distinct regions,
// plus a sorted list of all plan names. planIDToName maps plan UUID → plan name.
// The special key "" in the returned map contains all regions across all plans.
func BuildPlanRegionIndex(params []internal.ProvisioningParameters, planIDToName map[string]string) ([]string, map[string][]string) {
	planSet := make(map[string]struct{})
	// plan → region → present
	byPlan := make(map[string]map[string]struct{})

	for _, p := range params {
		name := planIDToName[p.PlanID]
		if name == "" {
			name = p.PlanID // fallback to raw ID
		}
		planSet[name] = struct{}{}

		region := ""
		if p.Parameters.Region != nil {
			region = *p.Parameters.Region
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
func FilterByPlan(params []internal.ProvisioningParameters, plan string, planIDToName map[string]string) []internal.ProvisioningParameters {
	result := params[:0:0]
	for _, p := range params {
		name := planIDToName[p.PlanID]
		if name == "" {
			name = p.PlanID
		}
		if name == plan {
			result = append(result, p)
		}
	}
	return result
}

// FilterByRegion returns only params where Parameters.Region matches the given region.
func FilterByRegion(params []internal.ProvisioningParameters, region string) []internal.ProvisioningParameters {
	result := params[:0:0]
	for _, p := range params {
		if p.Parameters.Region != nil && *p.Parameters.Region == region {
			result = append(result, p)
		}
	}
	return result
}
