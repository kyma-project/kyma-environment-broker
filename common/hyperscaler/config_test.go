package hyperscaler

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

		require.Empty(t, cfg.Rule)
	})

	t.Run("should read single values from env variables", func(t *testing.T) {
		err := os.Setenv("APP_HAP_RULE", `- aws`)
		require.NoError(t, err)

		// given
		var cfg Config
		err = envconfig.InitWithPrefix(&cfg, "APP_HAP")
		require.NoError(t, err)

		require.True(t, cfg.Rule.Contains("aws:*"))
	})
}
