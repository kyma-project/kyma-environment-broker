package rules

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRulesServiceFromFile(t *testing.T) {
	t.Run("should create RulesService from valid file", func(t *testing.T) {
		// given
		expectedRules := []string{"rule1", "rule2"}
		content := "rule:\n"
		for _, rule := range expectedRules {
			content += "- " + rule + "\n"
		}

		tmpfile := createTempFile(t, content)

		defer os.Remove(tmpfile)

		// when
		service, err := NewRulesServiceFromFile(tmpfile)

		// then
		require.NoError(t, err)
		require.NotNil(t, service)
	})

	t.Run("should return error when file path is empty", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("")

		// then
		require.Error(t, err)
		require.Nil(t, service)
		require.Equal(t, "No HAP rules file provided", err.Error())
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("nonexistent.yaml")

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

	t.Run("should return error when YAML file is corrupted", func(t *testing.T) {
		// given
		content := "rule:\n- rule1\n- rule2\ncorrupted_content"

		tmpfile := createTempFile(t, content)
		defer os.Remove(tmpfile)

		// when
		service, err := NewRulesServiceFromFile(tmpfile)

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

}

func createTempFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "test*.yaml")
	require.NoError(t, err)

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}
