package rules

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRulesServiceFromFile(t *testing.T) {
	t.Run("should create RulesService from valid file", func(t *testing.T) {
		// given
		content := `rule:
                      - rule1
                      - rule2`

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)

		defer os.Remove(tmpfile)

		// when
		service, err := NewRulesServiceFromFile(tmpfile, false, false, false)

		// then
		require.NoError(t, err)
		require.NotNil(t, service)
	})

	t.Run("should return error when file path is empty", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("", false, false, false)

		// then
		require.Error(t, err)
		require.Nil(t, service)
		require.Equal(t, "No HAP rules file path provided", err.Error())
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("nonexistent.yaml", false, false, false)

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
		service, err := NewRulesServiceFromFile(tmpfile, false, false, false)

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

}
