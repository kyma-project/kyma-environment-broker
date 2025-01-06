package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWhitelist(t *testing.T) {

	t.Run("should unmarshal from string representation", func(t *testing.T) {
		// given
		whitelist := Whitelist{}

		// when
		err := whitelist.Unmarshal("key1;key2")
		require.NoError(t, err)

		// then
		require.True(t, whitelist.Contains("key1"))
		require.True(t, whitelist.Contains("key2"))
		require.False(t, whitelist.Contains("key3"))
	})

	t.Run("should print all values", func(t *testing.T) {
		// given
		whitelist := Whitelist{"key1": struct{}{}, "key2": struct{}{}}

		// then
		require.Equal(t, "key1;key2;", whitelist.String())
		require.Equal(t, "key1;key2;", fmt.Sprint(whitelist))
	})
}
