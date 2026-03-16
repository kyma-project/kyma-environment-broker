package multiaccount

import runtimepkg "github.com/kyma-project/kyma-environment-broker/common/runtime"

type MultiAccountConfig struct {
	AllowedGlobalAccounts []string
	Limits                HyperscalerAccountLimits
	MinBindingsForGuard   int `envconfig:"default=0"`
}

type HyperscalerAccountLimits struct {
	Default int `envconfig:"default=999999"`

	AWS       int `envconfig:"optional"`
	GCP       int `envconfig:"optional"`
	Azure     int `envconfig:"optional"`
	OpenStack int `envconfig:"optional"`
	AliCloud  int `envconfig:"optional"`
}

func (c *MultiAccountConfig) IsEnabled() bool {
	return c != nil && len(c.AllowedGlobalAccounts) > 0
}

func (c *MultiAccountConfig) IsGlobalAccountAllowed(globalAccountID string) bool {
	if !c.IsEnabled() {
		return false
	}

	for _, ga := range c.AllowedGlobalAccounts {
		if ga == "*" || ga == globalAccountID {
			return true
		}
	}

	return false
}

func (c *MultiAccountConfig) LimitForProvider(providerType string) int {
	if c == nil {
		return 0
	}
	cp := runtimepkg.CloudProviderFromString(providerType)

	var limit int
	switch cp {
	case runtimepkg.AWS:
		limit = c.Limits.AWS
	case runtimepkg.GCP:
		limit = c.Limits.GCP
	case runtimepkg.Azure:
		limit = c.Limits.Azure
	case runtimepkg.SapConvergedCloud:
		limit = c.Limits.OpenStack
	case runtimepkg.Alicloud:
		limit = c.Limits.AliCloud
	default:
		limit = c.Limits.Default
	}

	if limit == 0 {
		limit = c.Limits.Default
	}

	// 0 means no limit (unlimited)
	if limit == 0 {
		return 999999
	}

	return limit
}
