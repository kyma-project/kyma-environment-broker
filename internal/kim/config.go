package kim

type Config struct {
	Enabled      bool     `envconfig:"default=false"` // if true, KIM will be used
	DryRun       bool     `envconfig:"default=true"`  // if true, only yamls are generated, no resources are created
	ViewOnly     bool     `envconfig:"default=true"`  // if true, provisioner will control the process
	Plans        []string `envconfig:"default=preview"`
	KimOnlyPlans []string `envconfig:"default=,"`
}

func (c *Config) IsEnabledForPlan(planName string) bool {
	if c.Enabled == false {
		return false
	}
	for _, plan := range c.Plans {
		if plan == planName {
			return true
		}
	}
	return false
}

func (c *Config) IsDrivenByKimOnly(planName string) bool {
	if !c.IsEnabledForPlan(planName) {
		return false
	}
	for _, plan := range c.KimOnlyPlans {
		if plan == planName {
			return true
		}
	}
	return false
}

func (c *Config) IsDrivenByKim(planName string) bool {
	return (c.IsEnabledForPlan(planName) && !c.ViewOnly && !c.DryRun) || c.IsDrivenByKimOnly(planName)
}
