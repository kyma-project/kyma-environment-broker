package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvergedCloudRegions_GetDefaultRegions(t *testing.T) {
	// given
	c := &OneForAllConvergedCloudRegionsProvider{}

	// when
	result := c.GetRegions()

	// then
	assert.Equal(t, []string{"eu-de-1"}, result)
}
