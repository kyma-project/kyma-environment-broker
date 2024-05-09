package subaccountsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const windowSize = 20 * 60 * 1000

func TestEventWindow(t *testing.T) {
	now := int64(0)
	ew := NewEventWindow(windowSize, func() int64 { return now })

	t.Run("should not return negative value", func(t *testing.T) {
		from := ew.GetNextFromTime()
		assert.Equal(t, int64(0), from)
	})

	t.Run("should return beginning of time when called first time, and sized window when called second time", func(t *testing.T) {
		now = 1709164800000 // 2024-12-28 00:00:00
		from := ew.GetNextFromTime()
		assert.Equal(t, int64(0), from)
		ew.UpdateToTime(now - 1000) // second to midnight
		ew.UpdateFromTime(from)
		assert.Equal(t, now-1000, ew.lastToTime)
		assert.Equal(t, from, ew.lastFromTime)
		now = 1709164800000 + 15*60*1000 // 2024-12-28 00:15:00
		from = ew.GetNextFromTime()
		assert.Equal(t, now-windowSize, from)
		ew.UpdateToTime(now - 1000) // 2024-12-28 00:14:59
		ew.UpdateFromTime(from)
		assert.Equal(t, now-1000, ew.lastToTime)
		assert.Equal(t, from, ew.lastFromTime)
	})
}
