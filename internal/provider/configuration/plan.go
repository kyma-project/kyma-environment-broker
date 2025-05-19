package configuration

import (
	"io"
	"strings"

	"gopkg.in/yaml.v2"
)

type PlanSpecifications struct {
	plans map[string]planSpecificationDTO
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
