package configuration

import (
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

	RegularMachines    []string `yaml:"regularMachines"`
	AdditionalMachines []string `yaml:"additionalMachines"`
	VolumeSizeGb       int      `yaml:"volumeSizeGb"`
	UpgradableToPlans  []string `yaml:"upgradableToPlans,omitempty"`
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
		if strings.ToLower(upgradablePlan) == strings.ToLower(to) {
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
		if strings.ToLower(target) != strings.ToLower(planName) {
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
