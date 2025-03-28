package rules

import (
	"fmt"
	"strconv"
)

const (
	PR_ATTR_NAME        = "PR"
	HR_ATTR_NAME        = "HR"
	EU_ATTR_NAME        = "EU"
	SHARED_ATTR_NAME    = "S"
	PR_SUFFIX_ATTR_NAME = "PR"
	HR_SUFFIX_ATTR_NAME = "HR"
)

var InputAttributes = []Attribute{
	{
		Name:        PR_ATTR_NAME,
		Description: "Platform Region",
		Setter:      setPlatformRegion,
		Getter:      func(r *Rule) string { return r.PlatformRegion },
		input:       true,
		output:      true,
		HasValue:    true,
	},
	{
		Name:        HR_ATTR_NAME,
		Description: "Hyperscaler Region",
		Setter:      setHyperscalerRegion,
		Getter:      func(r *Rule) string { return r.HyperscalerRegion },
		input:       true,
		output:      true,
		HasValue:    true,
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
	},
	{
		Name:        SHARED_ATTR_NAME,
		Description: "Shared",
		Setter:      setShared,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.Shared) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
	{
		Name:        PR_SUFFIX_ATTR_NAME,
		Description: "Platform Region suffix",
		Setter:      setPlatformRegionSuffix,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.PlatformRegionSuffix) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
	{
		Name:        HR_SUFFIX_ATTR_NAME,
		Description: "Platform Region suffix",
		Setter:      setHyperscalerRegionSuffix,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.HyperscalerRegionSuffix) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
}

func setShared(r *Rule, value string) (*Rule, error) {
	if r.Shared {
		return nil, fmt.Errorf("Shared already set")
	}

	r.ContainsOutputAttributes = true
	r.Shared = true

	return r, nil
}

func setPlatformRegionSuffix(r *Rule, value string) (*Rule, error) {
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

func (r *Rule) SetPlan(value string) (*Rule, error) {
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
