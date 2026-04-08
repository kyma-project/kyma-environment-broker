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

type OperationBlocklistEntryDTO struct {
	Message    string
	Attributes map[string]string
}

func (e *OperationBlocklistEntryDTO) UnmarshalYAML(value *yaml.Node) error {
	raw := value.Value
	if idx := strings.Index(raw, "#"); idx != -1 {
		raw = raw[:idx]
	}
	parts := strings.Split(raw, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.Trim(strings.TrimSpace(p), `"`))
	}
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("operation blocklist entry message is mandatory")
	}
	e.Message = parts[0]
	if len(parts) > 1 {
		e.Attributes = make(map[string]string)
		for _, attr := range parts[1:] {
			k, v, found := strings.Cut(attr, "=")
			if found {
				e.Attributes[k] = v
			}
		}
	}
	return nil
}

type OperationBlocklistDTO struct {
	Provision   OperationBlocklistEntryDTO `yaml:"provision"`
	Update      OperationBlocklistEntryDTO `yaml:"update"`
	PlanUpgrade OperationBlocklistEntryDTO `yaml:"planUpgrade"`
}

type planSpecificationDTO struct {
	// platform region -> list of hyperscaler regions
	Regions map[string][]string `yaml:"regions"`

	RegularMachines    []string               `yaml:"regularMachines"`
	AdditionalMachines []string               `yaml:"additionalMachines"`
	VolumeSizeGb       int                    `yaml:"volumeSizeGb"`
	UpgradableToPlans  []string               `yaml:"upgradableToPlans,omitempty"`
	OperationBlocklist *OperationBlocklistDTO `yaml:"operationBlocklist,omitempty"`
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

func (p *PlanSpecifications) OperationBlocklist(planName string) *OperationBlocklistDTO {
	plan, ok := p.plans[planName]
	if !ok {
		return nil
	}
	return plan.OperationBlocklist
}
