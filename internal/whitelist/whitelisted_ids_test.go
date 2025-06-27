package whitelist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWhitelistedIdsFromFile(t *testing.T) {
	// given/when
	d, err := ReadWhitelistedIdsFromFile("test/whitelist.yaml")

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, len(d))
	assert.Equal(t, struct{}{}, d["whitelisted-id"])
	assert.Equal(t, struct{}{}, d["another-whitelisted-id"])
}
