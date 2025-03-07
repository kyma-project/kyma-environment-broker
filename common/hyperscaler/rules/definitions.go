package rules

import (
	"fmt"
	"strconv"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

const (
	PR_ATTR_NAME     = "PR"
	HR_ATTR_NAME     = "HR"
	EU_ATTR_NAME     = "EU"
	SHARED_ATTR_NAME = "S"

	HYPERSCALER_LABEL = "hyperscalerType"
	EUACCESS_LABEL    = "euAccess"
	SHARED_LABEL      = "shared"

	ASTERISK                 = "*"
	ATTRIBUTE_WITH_VALUE     = "attr"
	SIGNATURE_ATTR_SEPARATOR = ":"
)

var InputAttributes = []Attribute{
	{
		Name:            PR_ATTR_NAME,
		Description:     "Platform Region",
		Setter:          setPlatformRegion,
		Getter:          func(r *Rule) string { return r.PlatformRegion },
		MatchableGetter: func(r *ProvisioningAttributes) string { return r.PlatformRegion },
		input:           true,
		output:          false,
		HasValue:        true,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			labels[HYPERSCALER_LABEL] += "_" + r.PlatformRegion
			return labels
		},
	},
	{
		Name:            HR_ATTR_NAME,
		Description:     "Hyperscaler Region",
		Setter:          setHyperscalerRegion,
		Getter:          func(r *Rule) string { return r.HyperscalerRegion },
		MatchableGetter: func(r *ProvisioningAttributes) string { return r.PlatformRegion },
		input:           true,
		output:          false,
		HasValue:        true,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			labels[HYPERSCALER_LABEL] += "_" + r.HyperscalerRegion
			return labels
		},
	},
}

var OutputAttributes = []Attribute{
	{
		Name:        EU_ATTR_NAME,
		Description: "EU Access",
		Setter:      setEuAccess,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.EuAccess) },
		input:       false,
		output:      true,
		HasValue:    false,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			if r.EuAccess {
				labels[EUACCESS_LABEL] = "true"
			}
			return labels
		},
	},
	{
		Name:        SHARED_ATTR_NAME,
		Description: "Shared",
		Setter:      setShared,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.Shared) },
		input:       false,
		output:      true,
		HasValue:    false,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			if r.Shared {
				labels[SHARED_LABEL] = "true"
			}

			return labels
		},
	},
}

var AllAttributes = append(InputAttributes, OutputAttributes...)

func setShared(r *Rule, value string) (*Rule, error) {
	if r.Shared {
		return nil, fmt.Errorf("Shared already set")
	}

	r.ContainsOutputAttributes = true
	r.Shared = true

	return r, nil
}

func setEuAccess(r *Rule, value string) (*Rule, error) {
	if r.EuAccess {
		return nil, fmt.Errorf("EuAccess already set")
	}
	r.ContainsOutputAttributes = true
	r.EuAccess = true

	return r, nil
}

func (r *Rule) SetPlan(value string, enabledPlans *broker.EnablePlans) (*Rule, error) {
	// validate that the plan is supported
	ok := enabledPlans.Contains(value)
	if !ok {
		return nil, fmt.Errorf("plan %s is not supported", value)
	}

	return r.SetPlanNoValidation(value)
}

func (r *Rule) SetPlanNoValidation(value string) (*Rule, error) {
	if value == "" {
		return nil, fmt.Errorf("plan is empty")
	}

	r.Plan = value
	r.Labels[HYPERSCALER_LABEL] = r.hyperscalerNameMappingFunction(value)

	return r, nil
}

func setPlatformRegion(r *Rule, value string) (*Rule, error) {
	if r.PlatformRegion != "" {
		return nil, fmt.Errorf("PlatformRegion already set")
	} else if value == "" {
		return nil, fmt.Errorf("PlatformRegion is empty")
	}

	r.ContainsInputAttributes = true
	r.PlatformRegion = value

	return r, nil
}

func setHyperscalerRegion(r *Rule, value string) (*Rule, error) {
	if r.HyperscalerRegion != "" {
		return nil, fmt.Errorf("HyperscalerRegion already set")
	} else if value == "" {
		return nil, fmt.Errorf("HyperscalerRegion is empty")
	}

	r.ContainsInputAttributes = true
	r.HyperscalerRegion = value

	return r, nil
}
