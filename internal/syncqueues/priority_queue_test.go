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

func TestPriorityQueueWithCallbacks(t *testing.T) {
	q := NewPriorityQueueWithCallbacksForSize(log, nil, 3)

	t.Run("should detect empty queue", func(t *testing.T) {
		assert.True(t, q.IsEmpty())
		e, ok := q.Extract()
		assert.False(t, ok)
		assert.Equal(t, QueueElement{}, e)
	})

	t.Run("should insert element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 0})
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert second element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert third element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e, ok := q.Extract()
		assert.True(t, ok)
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
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, 3, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid2, e.SubaccountID)
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid3, e.SubaccountID)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid4, e.SubaccountID)
		assert.True(t, q.IsEmpty())
	})
	t.Run("should extract empty element", func(t *testing.T) {
		e, ok := q.Extract()
		assert.False(t, ok)
		assert.Equal(t, "", e.SubaccountID)
		assert.Equal(t, "", e.BetaEnabled)
		assert.Equal(t, int64(0), e.ModifiedAt)
		assert.True(t, q.IsEmpty())
	})
}

var onInsertCalled = false
var onExtractCalled = false

func TestPriorityQueueWithEventHandlers(t *testing.T) {

	onInsertCallback := func(queueSize int) {
		assert.Equal(t, 1, queueSize)
		onInsertCalled = true
	}
	onExtractCallback := func(queueSize int, timeEnqueued int64) {
		assert.Equal(t, 0, queueSize)
		onExtractCalled = true
	}

	t.Run("should call onInsert but not onExtract", func(t *testing.T) {
		qWithOnInsert := NewPriorityQueueWithCallbacksForSize(log, &EventHandler{
			OnInsert: onInsertCallback,
		}, 4)
		onInsertCalled = false
		onExtractCalled = false
		qWithOnInsert.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		assert.True(t, onInsertCalled)
		assert.False(t, onExtractCalled)
		onInsertCalled = false
		e, ok := qWithOnInsert.Extract()
		assert.True(t, ok)
		assert.False(t, onInsertCalled)
		assert.False(t, onExtractCalled)
		assert.Equal(t, subaccountid2, e.SubaccountID)
	})

	t.Run("should call onExtract but not onInsert", func(t *testing.T) {
		qWithOnExtract := NewPriorityQueueWithCallbacksForSize(log, &EventHandler{
			OnExtract: onExtractCallback,
		}, 4)
		onInsertCalled = false
		onExtractCalled = false
		qWithOnExtract.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		assert.False(t, onInsertCalled)
		assert.False(t, onExtractCalled)
		onExtractCalled = false
		e, ok := qWithOnExtract.Extract()
		assert.True(t, ok)
		assert.False(t, onInsertCalled)
		assert.True(t, onExtractCalled)
		assert.Equal(t, subaccountid2, e.SubaccountID)
	})
	t.Run("should call onInsert and onExtract", func(t *testing.T) {
		qWithOnBoth := NewPriorityQueueWithCallbacksForSize(log, &EventHandler{
			OnInsert:  onInsertCallback,
			OnExtract: onExtractCallback,
		}, 4)

		onInsertCalled = false
		onExtractCalled = false
		qWithOnBoth.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		assert.True(t, onInsertCalled)
		assert.False(t, onExtractCalled)
		onInsertCalled = false
		e, ok := qWithOnBoth.Extract()
		assert.True(t, ok)
		assert.False(t, onInsertCalled)
		assert.True(t, onExtractCalled)
		assert.Equal(t, subaccountid2, e.SubaccountID)
	})
}
