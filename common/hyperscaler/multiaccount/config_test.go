package multiaccount

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitForProvider(t *testing.T) {
	t.Run("should return provider-specific limit when set", func(t *testing.T) {
		config := &MultiAccountConfig{
			Limits: HyperscalerAccountLimits{
				Default:   100,
				AWS:       180,
				GCP:       135,
				Azure:     135,
				OpenStack: 100,
				AliCloud:  100,
			},
		}

		assert.Equal(t, 180, config.LimitForProvider("aws"))
		assert.Equal(t, 135, config.LimitForProvider("gcp"))
		assert.Equal(t, 135, config.LimitForProvider("azure"))
		assert.Equal(t, 100, config.LimitForProvider("openstack"))
		assert.Equal(t, 100, config.LimitForProvider("alicloud"))
	})

	t.Run("should return default limit for unknown provider", func(t *testing.T) {
		config := &MultiAccountConfig{
			Limits: HyperscalerAccountLimits{
				Default:   250,
				AWS:       180,
				GCP:       135,
				Azure:     135,
				OpenStack: 100,
				AliCloud:  100,
			},
		}

		assert.Equal(t, 250, config.LimitForProvider("unknown-provider"))
		assert.Equal(t, 250, config.LimitForProvider("new-hyperscaler"))
	})

	t.Run("should return default limit when provider-specific limit is zero", func(t *testing.T) {
		config := &MultiAccountConfig{
			Limits: HyperscalerAccountLimits{
				Default:   150,
				AWS:       0,
				GCP:       135,
				Azure:     0,
				OpenStack: 100,
			},
		}

		assert.Equal(t, 150, config.LimitForProvider("aws"))
		assert.Equal(t, 135, config.LimitForProvider("gcp"))
		assert.Equal(t, 150, config.LimitForProvider("azure"))
		assert.Equal(t, 100, config.LimitForProvider("openstack"))
		assert.Equal(t, 150, config.LimitForProvider("alicloud"))
	})

	t.Run("should return 999999 when default is zero", func(t *testing.T) {
		config := &MultiAccountConfig{
			Limits: HyperscalerAccountLimits{
				Default: 0,
				AWS:     0,
				GCP:     0,
			},
		}

		assert.Equal(t, 999999, config.LimitForProvider("aws"))
		assert.Equal(t, 999999, config.LimitForProvider("gcp"))
		assert.Equal(t, 999999, config.LimitForProvider("azure"))
		assert.Equal(t, 999999, config.LimitForProvider("unknown"))
	})

	t.Run("should return 0 when config is nil", func(t *testing.T) {
		assert.Equal(t, 0, (*MultiAccountConfig)(nil).LimitForProvider("aws"))
		assert.Equal(t, 0, (*MultiAccountConfig)(nil).LimitForProvider("unknown"))
	})

	t.Run("should handle openstack provider name variants", func(t *testing.T) {
		config := &MultiAccountConfig{
			Limits: HyperscalerAccountLimits{
				Default:   100,
				OpenStack: 75,
			},
		}

		// All these variants map to SapConvergedCloud → OpenStack limit
		assert.Equal(t, 75, config.LimitForProvider("openstack"))
		assert.Equal(t, 75, config.LimitForProvider("sapconvergedcloud"))
		assert.Equal(t, 75, config.LimitForProvider("sap-converged-cloud"))
	})

	t.Run("should use default when OpenStack limit is not set", func(t *testing.T) {
		config := &MultiAccountConfig{
			Limits: HyperscalerAccountLimits{
				Default:   200,
				OpenStack: 0,
			},
		}

		assert.Equal(t, 200, config.LimitForProvider("openstack"))
		assert.Equal(t, 200, config.LimitForProvider("sapconvergedcloud"))
	})
}

func TestIsEnabled(t *testing.T) {
	t.Run("should return true when allowed global accounts is not empty", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-123"},
		}
		assert.True(t, config.IsEnabled())
	})

	t.Run("should return true when allowed global accounts has wildcard", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"*"},
		}
		assert.True(t, config.IsEnabled())
	})

	t.Run("should return true when allowed global accounts has multiple entries", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-1", "ga-2", "ga-3"},
		}
		assert.True(t, config.IsEnabled())
	})

	t.Run("should return false when allowed global accounts is empty", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{},
		}
		assert.False(t, config.IsEnabled())
	})

	t.Run("should return false when config is nil", func(t *testing.T) {
		assert.False(t, (*MultiAccountConfig)(nil).IsEnabled())
	})
}

func TestIsGlobalAccountAllowed(t *testing.T) {
	t.Run("should return true when GA is in allowed global accounts", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-123", "ga-456", "ga-789"},
		}
		assert.True(t, config.IsGlobalAccountAllowed("ga-123"))
		assert.True(t, config.IsGlobalAccountAllowed("ga-456"))
		assert.True(t, config.IsGlobalAccountAllowed("ga-789"))
	})

	t.Run("should return false when GA is not in allowed global accounts", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-123", "ga-456"},
		}
		assert.False(t, config.IsGlobalAccountAllowed("ga-999"))
		assert.False(t, config.IsGlobalAccountAllowed("ga-000"))
	})

	t.Run("should return true when allowed global accounts contains wildcard", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"*"},
		}
		assert.True(t, config.IsGlobalAccountAllowed("any-ga-id"))
		assert.True(t, config.IsGlobalAccountAllowed("another-ga"))
		assert.True(t, config.IsGlobalAccountAllowed("ga-123"))
	})

	t.Run("should return true when allowed global accounts has both wildcard and specific GAs", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-specific", "*"},
		}
		assert.True(t, config.IsGlobalAccountAllowed("ga-specific"))
		assert.True(t, config.IsGlobalAccountAllowed("any-other-ga"))
	})

	t.Run("should return false when allowed global accounts is empty", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{},
		}
		assert.False(t, config.IsGlobalAccountAllowed("ga-123"))
	})

	t.Run("should return false when config is nil", func(t *testing.T) {
		assert.False(t, (*MultiAccountConfig)(nil).IsGlobalAccountAllowed("ga-123"))
	})

	t.Run("should be case-sensitive for GA IDs", func(t *testing.T) {
		config := &MultiAccountConfig{
			AllowedGlobalAccounts: []string{"GA-123"},
		}
		assert.True(t, config.IsGlobalAccountAllowed("GA-123"))
		assert.False(t, config.IsGlobalAccountAllowed("ga-123"))
	})
}
