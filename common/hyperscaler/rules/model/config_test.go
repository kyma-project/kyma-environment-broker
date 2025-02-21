package model

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("should load rules from a file", func(t *testing.T) {
  
        // given
		// Create a temporary YAML file for testing
		expectedRules := []string{"rule1", "rule2"}
        content := "rule:\n"
  
        for _, rule := range expectedRules {
            content += "- " + rule + "\n"
        }
		tmpfile, err := os.CreateTemp("", "test*.yaml")
        require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatalf("Failed to close temp file: %v", err)
		}

        // when
		var config RulesConfig
		_, err = config.Load(tmpfile.Name())
        require.NoError(t, err)

        // then
		for i, rule := range config.Rules {
			if rule != expectedRules[i] {
				t.Errorf("Expected rule %s, got %s", expectedRules[i], rule)
			}
		}
	})
}
