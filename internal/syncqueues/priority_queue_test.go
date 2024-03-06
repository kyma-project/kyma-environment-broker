package syncqueues

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	subaccountid1 = "sa-1"
	subaccountid2 = "sa-2"
	subaccountid3 = "sa-3"
	subaccountid4 = "sa-4"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func TestPriorityQueue(t *testing.T) {
	q := NewPriorityQueueForTests(log, 3)

	t.Run("should insert element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 0})
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert second element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert third element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid2, e.SubaccountID)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should not insert outdated element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 1})
		assert.Equal(t, 1, q.size)
	})
	t.Run("should update element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 3})
		assert.Equal(t, 1, q.size)
	})
	t.Run("should update element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 3})
		assert.Equal(t, 2, q.size)
	})
	t.Run("should update element again", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 5})
		assert.Equal(t, 2, q.size)
	})
	t.Run("should insert element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "true", ModifiedAt: 4})
		assert.Equal(t, 3, q.size)
		assert.Equal(t, 3, cap(q.elements))
	})
	t.Run("should resize queue and insert element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid4, BetaEnabled: "true", ModifiedAt: 6})
		assert.Equal(t, 4, q.size)
		assert.Equal(t, 6, cap(q.elements))
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, 3, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid2, e.SubaccountID)
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid3, e.SubaccountID)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid4, e.SubaccountID)
		assert.True(t, q.IsEmpty())
	})
	t.Run("should extract empty element", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, "", e.SubaccountID)
		assert.Equal(t, "", e.BetaEnabled)
		assert.Equal(t, int64(0), e.ModifiedAt)
		assert.True(t, q.IsEmpty())
	})
}
