package rules

import (
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/require"
)

func TestNewRulesServiceFromFile(t *testing.T) {
	t.Run("should create RulesService from valid file ane parse simple rules", func(t *testing.T) {
		// given
		content := `rule:
                      - rule1
                      - rule2`

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)

		defer os.Remove(tmpfile)

		// when
		enabledPlans := &broker.EnablePlans{"rule1", "rule2"}
		service, err := NewRulesServiceFromFile(tmpfile, enabledPlans)

		// then
		require.NoError(t, err)
		require.NotNil(t, service)

		require.Equal(t, 2, len(service.ParsedRuleset.Results))
		for _, result := range service.ParsedRuleset.Results {
			require.False(t, result.HasErrors())
		}
	})

	t.Run("should return error when file path is empty", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("", &broker.EnablePlans{})

		// then
		require.Error(t, err)
		require.Nil(t, service)
		require.Equal(t, "No HAP rules file path provided", err.Error())
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("nonexistent.yaml", &broker.EnablePlans{})

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

	t.Run("should return error when YAML file is corrupted", func(t *testing.T) {
		// given
		content := "corrupted_content"

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)
		defer os.Remove(tmpfile)

		// when
		service, err := NewRulesServiceFromFile(tmpfile, &broker.EnablePlans{})

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

}
