package analytics

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type fieldBehavior int

const (
	behaviorValue fieldBehavior = iota // emit field value as string (default)
	behaviorSkip                       // ignore field entirely
	behaviorCount                      // emit slice/array length as value
)

// provisioningFieldConfig controls per-field behavior for ProvisioningParametersDTO.
// Fields not listed default to behaviorValue.
var provisioningFieldConfig = map[string]fieldBehavior{
	"Zones":                     behaviorSkip,
	"TargetSecret":              behaviorSkip,
	"Kubeconfig":                behaviorSkip,
	"ShootName":                 behaviorSkip,
	"ShootDomain":               behaviorSkip,
	"RuntimeAdministrators":     behaviorCount,
	"AdditionalWorkerNodePools": behaviorCount,
}

// updatingFieldConfig controls per-field behavior for UpdatingParametersDTO.
var updatingFieldConfig = map[string]fieldBehavior{
	"RuntimeAdministrators":     behaviorCount,
	"AdditionalWorkerNodePools": behaviorCount,
}

// walkFields reflects over a struct, applies fieldConfig, and populates counts:
//
//	counts[fieldName][value] = occurrenceCount
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

		behavior, ok := config[field.Name]
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

		if _, ok := counts[field.Name]; !ok {
			counts[field.Name] = make(map[string]int)
		}
		counts[field.Name][value]++
	}
}

// AggregateProvisioning computes parameter usage stats from a slice of ProvisioningParameters.
func AggregateProvisioning(params []internal.ProvisioningParameters) ParameterStats {
	counts := make(map[string]map[string]int)
	total := len(params)
	for _, p := range params {
		walkFields(p.Parameters, provisioningFieldConfig, counts)
	}
	return toParameterStats(counts, total)
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
	sort.Slice(result, func(i, j int) bool {
		return result[i].SetCount > result[j].SetCount
	})
	return ParameterStats{Parameters: result}
}

// BuildDistributions computes value breakdowns for selected distribution fields.
func BuildDistributions(params []internal.ProvisioningParameters) []DistributionStat {
	distributionFields := []string{"MachineType", "Region", "AutoScalerMin", "AutoScalerMax"}
	counts := make(map[string]map[string]int)
	for _, p := range params {
		walkFields(p.Parameters, provisioningFieldConfig, counts)
	}
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
