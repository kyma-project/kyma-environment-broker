package rules

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("should load rules from a file", func(t *testing.T) {
		// given
		content := `rule:
                      - rule1
                      - rule2`

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		// when
		var config RulesConfig
		err = config.Load(tmpfile)
		require.NoError(t, err)

		// then
		require.Equal(t, "rule1", config.Rules[0])
		require.Equal(t, "rule2", config.Rules[1])
	})
}
