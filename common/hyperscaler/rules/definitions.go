package rules

import (
	"fmt"
	"strconv"
)

const (
	PrAttrName       = "PR"
	HrAttrName       = "HR"
	EuAttrName       = "EU"
	SharedAttrName   = "S"
	PrSuffixAttrName = "PR"
	HrSuffixAttrName = "HR"
)

var InputAttributes = []Attribute{
	{
		Name:        PrAttrName,
		Description: "Platform Region",
		Setter:      setPlatformRegion,
		Getter:      func(r *Rule) string { return r.PlatformRegion },
		input:       true,
		output:      true,
		HasValue:    true,
	},
	{
		Name:        HrAttrName,
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
		Name:        EuAttrName,
		Description: "EU Access",
		Setter:      setEuAccess,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.EuAccess) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
	{
		Name:        SharedAttrName,
		Description: "Shared",
		Setter:      setShared,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.Shared) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
	{
		Name:        PrSuffixAttrName,
		Description: "Platform Region suffix",
		Setter:      setPlatformRegionSuffix,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.PlatformRegionSuffix) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
	{
		Name:        HrSuffixAttrName,
		Description: "Platform Region suffix",
		Setter:      setHyperscalerRegionSuffix,
		Getter:      func(r *Rule) string { return strconv.FormatBool(r.HyperscalerRegionSuffix) },
		input:       false,
		output:      true,
		HasValue:    false,
	},
}

func setShared(r *Rule, value string) error {
	if r.Shared {
		return fmt.Errorf("Shared already set")
	}

	r.ContainsOutputAttributes = true
	r.Shared = true

	return nil
}

func setPlatformRegionSuffix(r *Rule, value string) error {
	if r.PlatformRegionSuffix {
		return fmt.Errorf("PlatformRegionSuffix already set")
	}

	r.ContainsOutputAttributes = true
	r.PlatformRegionSuffix = true

	return nil
}

func setHyperscalerRegionSuffix(r *Rule, value string) error {
	if r.HyperscalerRegionSuffix {
		return fmt.Errorf("HyperscalerRegionSuffix already set")
	}

	r.ContainsOutputAttributes = true
	r.HyperscalerRegionSuffix = true

	return nil
}

func setEuAccess(r *Rule, value string) error {
	if r.EuAccess {
		return fmt.Errorf("EuAccess already set")
	}
	r.ContainsOutputAttributes = true
	r.EuAccess = true

	return nil
}

func (r *Rule) SetPlan(value string) (*Rule, error) {
	if value == "" {
		return nil, fmt.Errorf("plan name is empty")
	}

	r.Plan = value
	return r, nil
}

func setPlatformRegion(r *Rule, value string) error {
	if r.PlatformRegion != "" {
		return fmt.Errorf("PlatformRegion already set")
	} else if value == "" {
		return fmt.Errorf("PlatformRegion is empty")
	}

	r.ContainsInputAttributes = true
	r.PlatformRegion = value

	return nil
}

func setHyperscalerRegion(r *Rule, value string) error {
	if r.HyperscalerRegion != "" {
		return fmt.Errorf("HyperscalerRegion already set")
	} else if value == "" {
		return fmt.Errorf("HyperscalerRegion is empty")
	}

	r.ContainsInputAttributes = true
	r.HyperscalerRegion = value

	return nil
}
