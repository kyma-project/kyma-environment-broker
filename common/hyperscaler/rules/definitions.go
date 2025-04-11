package rules

import (
	"fmt"
	"strconv"
)

const (
	PlatformRegionAttributeName    = "PR"
	HyperscalerRegionAttributeName = "HR"
	EUAccessAttributeName          = "EU"
	SharedAttributeName            = "S"
	PlatformRegionSuffix           = "PR"
	HyperscalerRegionSuffix        = "HR"
)

var InputAttributes = []Attribute{
	{
		Name:     PlatformRegionAttributeName,
		Setter:   setPlatformRegion,
		Getter:   func(r *Rule) string { return r.PlatformRegion },
		HasValue: true,
	},
	{
		Name:     HyperscalerRegionAttributeName,
		Setter:   setHyperscalerRegion,
		Getter:   func(r *Rule) string { return r.HyperscalerRegion },
		HasValue: true,
	},
}

var OutputAttributes = []Attribute{
	{
		Name:     EUAccessAttributeName,
		Setter:   setEuAccess,
		Getter:   func(r *Rule) string { return strconv.FormatBool(r.EuAccess) },
		HasValue: false,
	},
	{
		Name:     SharedAttributeName,
		Setter:   setShared,
		Getter:   func(r *Rule) string { return strconv.FormatBool(r.Shared) },
		HasValue: false,
	},
	{
		Name:     PlatformRegionSuffix,
		Setter:   setPlatformRegionSuffix,
		Getter:   func(r *Rule) string { return strconv.FormatBool(r.PlatformRegionSuffix) },
		HasValue: false,
	},
	{
		Name:     HyperscalerRegionSuffix,
		Setter:   setHyperscalerRegionSuffix,
		Getter:   func(r *Rule) string { return strconv.FormatBool(r.HyperscalerRegionSuffix) },
		HasValue: false,
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
