package rules

import (
	"fmt"
	"strconv"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

const (
	PR_ATTR_NAME        = "PR"
	HR_ATTR_NAME        = "HR"
	EU_ATTR_NAME        = "EU"
	SHARED_ATTR_NAME    = "S"
	PR_SUFFIX_ATTR_NAME = "PR"
	HR_SUFFIX_ATTR_NAME = "HR"

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
		output:          true,
		HasValue:        true,
		ApplyLabel: func(r *Rule, provisioningAttributes *ProvisioningAttributes, labels map[string]string) map[string]string {
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
		output:          true,
		HasValue:        true,
		ApplyLabel: func(r *Rule, provisioningAttributes *ProvisioningAttributes, labels map[string]string) map[string]string {
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
		ApplyLabel: func(r *Rule, provisioningAttributes *ProvisioningAttributes, labels map[string]string) map[string]string {
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
		ApplyLabel: func(r *Rule, provisioningAttributes *ProvisioningAttributes, labels map[string]string) map[string]string {
			if r.Shared {
				labels[SHARED_LABEL] = "true"
			}

			return labels
		},
	},
	{
		Name:        PR_SUFFIX_ATTR_NAME,
		Description: "Platform Region suffix",
		Setter:      setPlatoformRegionSuffix,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.PlatformRegionSuffix) },
		input:       false,
		output:      true,
		HasValue:    false,
		ApplyLabel: func(r *Rule, provisioningAttributes *ProvisioningAttributes, labels map[string]string) map[string]string {
			if r.PlatformRegionSuffix {
				if provisioningAttributes.PlatformRegion != "" {
					labels[HYPERSCALER_LABEL] += "_" + provisioningAttributes.PlatformRegion
				} else {
					labels[HYPERSCALER_LABEL] += "_<PR>"
				}
			}
			return labels
		},
	},
	{
		Name:        HR_SUFFIX_ATTR_NAME,
		Description: "Platform Region suffix",
		Setter:      setHyperscalerRegionSuffix,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.HyperscalerRegionSuffix) },
		input:       false,
		output:      true,
		HasValue:    false,
		ApplyLabel: func(r *Rule, provisioningAttributes *ProvisioningAttributes, labels map[string]string) map[string]string {
			if r.HyperscalerRegionSuffix {
				if provisioningAttributes.HyperscalerRegion != "" {
					labels[HYPERSCALER_LABEL] += "_" + provisioningAttributes.HyperscalerRegion
				} else {
					labels[HYPERSCALER_LABEL] += "_<HR>"
				}
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

func setPlatoformRegionSuffix(r *Rule, value string) (*Rule, error) {
	if r.PlatformRegionSuffix {
		return nil, fmt.Errorf("PlatformRegionSuffix already set")
	}

	r.ContainsOutputAttributes = true
	r.PlatformRegionSuffix = true

	return r, nil
}

func setHyperscalerRegionSuffix(r *Rule, value string) (*Rule, error) {
	if r.HyperscalerRegionSuffix {
		return nil, fmt.Errorf("HyperscalerRegionSuffix already set")
	}

	r.ContainsOutputAttributes = true
	r.HyperscalerRegionSuffix = true

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
