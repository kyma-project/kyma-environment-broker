package config_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kymaTemplateConfigKey = "kyma-template"
)

func TestValidate(t *testing.T) {
	// setup
	cfgValidator := config.NewConfigMapKeysValidator()

	t.Run("should validate whether config contains required fields", func(t *testing.T) {
		// given
		cfgString := `kyma-template: ""`

		// when
		err := cfgValidator.Validate(kymaTemplateConfigKey, cfgString)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error indicating missing required fields", func(t *testing.T) {
		// given
		cfgString := `optional-field: "optional"`

		// when
		err := cfgValidator.Validate(kymaTemplateConfigKey, cfgString)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), kymaTemplateConfigKey)
	})
}
