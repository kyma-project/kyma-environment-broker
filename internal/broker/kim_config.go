package broker

import "strings"

type KimConfig struct {
	Enabled      bool     `envconfig:"default=false"` // if true, KIM will be used
	DryRun       bool     `envconfig:"default=true"`  // if true, only yamls are generated, no resources are created
	ViewOnly     bool     `envconfig:"default=true"`  // if true, provisioner will control the process
	Plans        []string `envconfig:"default=preview"`
	KimOnlyPlans []string `envconfig:"default=,"`
}

func (c *KimConfig) IsEnabledForPlan(planName string) bool {
	if c.Enabled == false {
		return false
	}
	for _, plan := range c.Plans {
		if strings.TrimSpace(plan) == planName {
			return true
		}
	}
	return false
}

func (c *KimConfig) IsDrivenByKimOnly(planName string) bool {
	if !c.IsEnabledForPlan(planName) {
		return false
	}
	for _, plan := range c.KimOnlyPlans {
		if strings.TrimSpace(plan) == planName {
			return true
		}
	}
	return false
}

func (c *KimConfig) IsPlanIdDrivenByKimOnly(planID string) bool {
	planName := PlanIDsMapping[planID]
	return c.IsDrivenByKimOnly(planName)
}

func (c *KimConfig) IsPlanIdDrivenByKim(planID string) bool {
	planName := PlanIDsMapping[planID]
	return c.IsDrivenByKim(planName)
}

func (c *KimConfig) IsDrivenByKim(planName string) bool {
	return (c.IsEnabledForPlan(planName) && !c.ViewOnly && !c.DryRun) || c.IsDrivenByKimOnly(planName)
}

func (c *KimConfig) IsEnabledForPlanID(planID string) bool {
	planName := PlanIDsMapping[planID]
	return c.IsEnabledForPlan(planName)
}
