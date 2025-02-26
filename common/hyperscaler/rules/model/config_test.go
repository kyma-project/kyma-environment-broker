package model

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("should load rules from a file", func(t *testing.T) {

		// given
		expectedRules := []string{"rule1", "rule2"}
		content := "rule:\n"

		for _, rule := range expectedRules {
			content += "- " + rule + "\n"
		}
		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)
		defer os.Remove(tmpfile)

		// when
		var config RulesConfig
		_, err = config.Load(tmpfile)
		require.NoError(t, err)

		// then
		for i, rule := range config.Rules {
			if rule != expectedRules[i] {
				t.Errorf("Expected rule %s, got %s", expectedRules[i], rule)
			}
		}
	})
}
