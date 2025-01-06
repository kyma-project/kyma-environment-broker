package hap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
)

func TestHyperscalerConfigs(t *testing.T) {

	t.Run("should read default values from env variables", func(t *testing.T) {
		// given
		var cfg Config
		err := envconfig.InitWithPrefix(&cfg, "APP_HAP")
		require.NoError(t, err)

		require.True(t, cfg.SharedSecretPlans.Contains("trial"))
		require.True(t, cfg.SharedSecretPlans.Contains("sap-converged-cloud"))

		require.False(t, cfg.SharedSecretRegions.Contains("eu1"))
		require.False(t, cfg.SharedSecretRegions.Contains("eu2"))
	})

	t.Run("should read single values from env variables", func(t *testing.T) {
		err := os.Setenv("APP_HAP_SHARED_SECRET_PLANS", "aws")
		require.NoError(t, err)

		err = os.Setenv("APP_HAP_SHARED_SECRET_REGIONS", "eu1;eu3")
		require.NoError(t, err)

		// given
		var cfg Config
		err = envconfig.InitWithPrefix(&cfg, "APP_HAP")
		require.NoError(t, err)

		require.True(t, cfg.SharedSecretPlans.Contains("aws"))
		require.False(t, cfg.SharedSecretPlans.Contains("azure"))

		require.True(t, cfg.SharedSecretRegions.Contains("eu1"))
		require.False(t, cfg.SharedSecretRegions.Contains("eu2"))
		require.True(t, cfg.SharedSecretRegions.Contains("eu3"))
	})
}
