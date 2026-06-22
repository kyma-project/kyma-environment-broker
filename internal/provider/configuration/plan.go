package configuration

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type PlanSpecifications struct {
	plans map[string]planSpecificationDTO
}

func NewPlanSpecificationsFromFile(filePath string) (*PlanSpecifications, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Use the existing function to parse the specifications
	return NewPlanSpecifications(file)
}

func NewPlanSpecifications(r io.Reader) (*PlanSpecifications, error) {
	spec := &PlanSpecifications{
		plans: make(map[string]planSpecificationDTO),
	}

	dto := PlanSpecificationsDTO{}
	d := yaml.NewDecoder(r)
	err := d.Decode(dto)

	for key, plan := range dto {
		planNames := strings.Split(key, ",")
		for _, planName := range planNames {
			spec.plans[planName] = plan
		}
	}

	return spec, err
}

type PlanSpecificationsDTO map[string]planSpecificationDTO

type planSpecificationDTO struct {
	// platform region -> list of hyperscaler regions
	Regions map[string][]string `yaml:"regions"`

	RegularMachines      []string `yaml:"regularMachines"`
	AdditionalMachines   []string `yaml:"additionalMachines"`
	InternalOnlyMachines []string `yaml:"internalOnlyMachines,omitempty"`
	VolumeSizeGb         int      `yaml:"volumeSizeGb"`
	UpgradableToPlans    []string `yaml:"upgradableToPlans,omitempty"`
}

func (p *PlanSpecifications) Regions(planName string, platformRegion string) []string {
	plan, ok := p.plans[planName]
	if !ok {
		return []string{}
	}

	regions, ok := plan.Regions[platformRegion]
	if !ok {
		defaultRegions, found := plan.Regions["default"]
		if found {
			return defaultRegions
		}
		return []string{}
	}

	return regions
}

func (p *PlanSpecifications) AllRegionsByPlan() map[string][]string {
	planRegions := map[string][]string{}
	for planName, plan := range p.plans {
		for _, regions := range plan.Regions {
			planRegions[planName] = append(planRegions[planName], regions...)
		}
	}
	return planRegions

}

func (p *PlanSpecifications) RegularMachines(planName string) []string {
	plan, ok := p.plans[planName]
	if !ok {
		return []string{}
	}
	return plan.RegularMachines
}

func (p *PlanSpecifications) AdditionalMachines(planName string) []string {
	plan, ok := p.plans[planName]
	if !ok {
		return []string{}
	}
	return plan.AdditionalMachines
}

func (p *PlanSpecifications) IsInternalOnlyMachine(planName, machineType string) bool {
	plan, ok := p.plans[planName]
	if !ok {
		return false
	}
	for _, entry := range plan.InternalOnlyMachines {
		if entry == "" {
			continue
		}
		if strings.HasPrefix(machineType, entry) {
			return true
		}
	}
	return false
}

func (p *PlanSpecifications) DefaultVolumeSizeGb(planName string) (int, bool) {
	plan, ok := p.plans[planName]
	if !ok {
		return 0, false
	}
	if plan.VolumeSizeGb == 0 {
		return 0, false
	}
	return plan.VolumeSizeGb, true
}

func (p *PlanSpecifications) IsUpgradableBetween(from, to string) bool {
	plan, ok := p.plans[from]
	if !ok {
		return false
	}
	for _, upgradablePlan := range plan.UpgradableToPlans {
		if strings.EqualFold(upgradablePlan, to) {
			return true
		}
	}
	return false
}

func (p *PlanSpecifications) IsUpgradable(planName string) bool {
	plan, ok := p.plans[planName]
	if !ok {
		return false
	}
	numberOfTargetPlans := 0
	for _, target := range plan.UpgradableToPlans {
		if !strings.EqualFold(target, planName) {
			numberOfTargetPlans++
		}
	}
	return numberOfTargetPlans > 0
}

func (p *PlanSpecifications) DefaultMachineType(planName string) string {
	regularMachines := p.RegularMachines(planName)
	if len(regularMachines) == 0 {
		return ""
	}
	return regularMachines[0]
}

// ValidateInternalOnlyMachines returns warning messages for misconfigured internalOnlyMachines entries:
// - redundant entries already covered by a prefix in the same list
// - entries that don't match any machine in regularMachines or additionalMachines
func (p *PlanSpecifications) ValidateInternalOnlyMachines() []string {
	var warnings []string
	for planName, plan := range p.plans {
		allMachines := append(plan.RegularMachines, plan.AdditionalMachines...)
		for i, entry := range plan.InternalOnlyMachines {
			for j, other := range plan.InternalOnlyMachines {
				if i != j && strings.HasPrefix(entry, other) && entry != other {
					warnings = append(warnings, fmt.Sprintf("internalOnlyMachines entry %q is redundant in plan %q — already covered by prefix %q", entry, planName, other))
					break
				}
			}
		}
		for _, entry := range plan.InternalOnlyMachines {
			matched := false
			for _, machine := range allMachines {
				if strings.HasPrefix(machine, entry) {
					matched = true
					break
				}
			}
			if !matched {
				warnings = append(warnings, fmt.Sprintf("internalOnlyMachines entry %q in plan %q does not match any machine type in regularMachines or additionalMachines", entry, planName))
			}
		}
	}
	return warnings
}
