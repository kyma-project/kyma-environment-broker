package analytics

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type fieldBehavior int

const (
	behaviorValue fieldBehavior = iota // emit field value as string (default)
	behaviorSkip                       // ignore field entirely
	behaviorCount                      // emit slice/array length as value
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
}

// updatingFieldConfig controls per-field behavior for UpdatingParametersDTO.
var updatingFieldConfig = map[string]fieldBehavior{
	"administrators":            behaviorCount,
	"additionalWorkerNodePools": behaviorCount,
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

		// Dereference pointers; skip nil
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		// Skip zero/empty values
		if fv.IsZero() {
			continue
		}

		var value string
		switch behavior {
		case behaviorCount:
			if fv.Kind() == reflect.Slice || fv.Kind() == reflect.Array {
				value = fmt.Sprintf("%d", fv.Len())
			} else {
				continue
			}
		default: // behaviorValue
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
				// struct (e.g. OIDCConnectDTO dereferenced) — treat as set/present
				value = "set"
			}
		}

		if _, ok := counts[jsonName]; !ok {
			counts[jsonName] = make(map[string]int)
		}
		counts[jsonName][value]++
	}
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

