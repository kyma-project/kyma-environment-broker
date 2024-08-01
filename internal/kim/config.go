package kim

type Config struct {
	Enabled      bool     `envconfig:"default=false"`
	DryRun       bool     `envconfig:"default=true"`
	ViewOnly     bool     `envconfig:"default=true"`
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

func (c *Config) DrivenByKimOnly(planName string) bool {
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
