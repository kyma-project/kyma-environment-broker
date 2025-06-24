package config

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigMapConverter(t *testing.T) {
	// given
	converter := NewConfigMapConverter()

	t.Run("should convert to struct", func(t *testing.T) {
		// given
		expectedValue := "test"
		var cfg internal.ConfigForPlan
		cfgStr := "kyma-template: test"

		// when
		require.NoError(t, converter.Convert(cfgStr, &cfg))

		// then
		assert.Equal(t, expectedValue, cfg.KymaTemplate)
	})

	t.Run("should convert to map[string]any", func(t *testing.T) {
		// given
		expectedMap := map[string]any{"kyma-template": "test"}
		var cfg map[string]any
		cfgStr := "kyma-template: test"

		// when
		require.NoError(t, converter.Convert(cfgStr, &cfg))

		// then
		assert.Equal(t, expectedMap, cfg)
	})
}
