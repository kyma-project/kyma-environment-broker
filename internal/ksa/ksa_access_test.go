package ksa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsKSARestrictedAccess(t *testing.T) {
	t.Run("WithRestrictedRegion_ReturnsTrue", func(t *testing.T) {
		result := IsKSARestrictedAccess("cf-sa30")
		assert.True(t, result)
	})

	t.Run("WithNonRestrictedRegion_ReturnsFalse", func(t *testing.T) {
		result := IsKSARestrictedAccess("non-restricted-region")
		assert.False(t, result)
	})

	t.Run("WithEmptyRegion_ReturnsFalse", func(t *testing.T) {
		result := IsKSARestrictedAccess("")
		assert.False(t, result)
	})
}
