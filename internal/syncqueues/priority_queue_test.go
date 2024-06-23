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

var log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

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
		// queue is [{subaccountid1, true, 0}]
		// idx is {subaccountid1: 0}
		assert.Equal(t, 1, q.size)
		_, _ = q.Extract()
		assert.True(t, q.IsEmpty())
	})

	t.Run("should insert two elements and extract element with minimal value of ModifiedAt", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 0})
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		// queue is [{subaccountid1, true, 0}, {subaccountid2, false, 1}]
		// idx is {subaccountid1: 0, subaccountid2: 1}
		assert.Equal(t, 2, q.size)
		e, ok := q.Extract()
		// queue is [{subaccountid2, false, 1}]
		// idx is {subaccountid2: 0}

		assert.True(t, ok)
		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, int64(0), e.ModifiedAt)
		assert.Equal(t, "true", e.BetaEnabled)
		assert.Equal(t, 1, q.size)
		_, _ = q.Extract()
		assert.True(t, q.IsEmpty())
	})

	t.Run("should insert three elements and extract element with minimal value of ModifiedAt", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 0})
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		assert.Equal(t, 3, q.size)
		e, ok := q.Extract()
		assert.True(t, ok)

		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, 2, q.size)
		assert.Equal(t, "true", e.BetaEnabled)
		assert.Equal(t, int64(0), e.ModifiedAt)
		_, _ = q.Extract()
		_, _ = q.Extract()
		assert.True(t, q.IsEmpty())
	})

	t.Run("should not insert outdated element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 1})
		// queue is [{subaccountid3, true, 2}]
		// idx is {subaccountid3: 0}
		assert.Equal(t, 1, q.size)
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid3, e.SubaccountID)
		assert.Equal(t, int64(2), e.ModifiedAt)
		assert.Equal(t, "true", e.BetaEnabled)
		assert.True(t, q.IsEmpty())
	})

	t.Run("should update element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
		// queue is [{subaccountid3, true, 3}]
		// idx is {subaccountid3: 0}
		assert.Equal(t, 1, q.size)
		e, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid3, e.SubaccountID)
		assert.Equal(t, int64(3), e.ModifiedAt)
		assert.Equal(t, "false", e.BetaEnabled)
		assert.True(t, q.IsEmpty())
	})

	t.Run("should insert, update, insert and update again and insert again", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 4})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 6})
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "true", ModifiedAt: 5})

		// queue is [{subaccountid1, true, 4}, {subaccountid2, true, 5}, {subaccountid3, true, 6}]
		// idx is {subaccountid1: 0, subaccountid2: 1, subaccountid3: 2}
		assert.Equal(t, 3, q.size)
		assert.Equal(t, 3, cap(q.elements))
		// extract e1
		e1, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid1, e1.SubaccountID)
		assert.Equal(t, int64(4), e1.ModifiedAt)
		assert.Equal(t, "true", e1.BetaEnabled)
		// extract e2
		e2, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid2, e2.SubaccountID)
		assert.Equal(t, int64(5), e2.ModifiedAt)
		assert.Equal(t, "true", e2.BetaEnabled)
		// extract e3
		e3, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid3, e3.SubaccountID)
		assert.Equal(t, int64(6), e3.ModifiedAt)
		assert.Equal(t, "true", e3.BetaEnabled)
		assert.True(t, q.IsEmpty())
	})

	t.Run("should resize queue and insert element", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 4})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 6})
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "true", ModifiedAt: 5})
		q.Insert(QueueElement{SubaccountID: subaccountid4, BetaEnabled: "true", ModifiedAt: 7})

		// queue is [{subaccountid1, true, 4}, {subaccountid2, true, 5}, {subaccountid3, true, 6}, {subaccountid4, true, 7}]
		// idx is {subaccountid1: 0, subaccountid2: 1, subaccountid3: 2, subaccountid4: 3}

		assert.Equal(t, 4, q.size)

		// extract e1
		e1, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid1, e1.SubaccountID)
		assert.Equal(t, int64(4), e1.ModifiedAt)
		assert.Equal(t, "true", e1.BetaEnabled)
		// extract e2
		e2, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid2, e2.SubaccountID)
		assert.Equal(t, int64(5), e2.ModifiedAt)
		assert.Equal(t, "true", e2.BetaEnabled)
		// extract e3
		e3, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid3, e3.SubaccountID)
		assert.Equal(t, int64(6), e3.ModifiedAt)
		assert.Equal(t, "true", e3.BetaEnabled)
		// extract e4
		e4, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid4, e4.SubaccountID)
		assert.Equal(t, int64(7), e4.ModifiedAt)
		assert.Equal(t, "true", e4.BetaEnabled)
		assert.True(t, q.IsEmpty())
	})

	t.Run("should not update element again", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
		q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 4})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 6})
		q.Insert(QueueElement{SubaccountID: subaccountid2, BetaEnabled: "true", ModifiedAt: 5})
		q.Insert(QueueElement{SubaccountID: subaccountid4, BetaEnabled: "true", ModifiedAt: 7})
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})

		// queue is [{subaccountid1, true, 4}, {subaccountid2, true, 5}, {subaccountid3, true, 6}, {subaccountid4, true, 7}]
		// idx is {subaccountid1: 0, subaccountid2: 1, subaccountid3: 2, subaccountid4: 3}

		// extract e1
		e1, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid1, e1.SubaccountID)
		assert.Equal(t, int64(4), e1.ModifiedAt)
		assert.Equal(t, "true", e1.BetaEnabled)
		// extract e2
		e2, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid2, e2.SubaccountID)
		assert.Equal(t, int64(5), e2.ModifiedAt)
		assert.Equal(t, "true", e2.BetaEnabled)
		// extract e3
		e3, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid3, e3.SubaccountID)
		assert.Equal(t, int64(6), e3.ModifiedAt)
		assert.Equal(t, "true", e3.BetaEnabled)
		// extract e4
		e4, ok := q.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountid4, e4.SubaccountID)
		assert.Equal(t, int64(7), e4.ModifiedAt)
		assert.Equal(t, "true", e4.BetaEnabled)
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

func TestUpdateScenario1(t *testing.T) {
	q := NewPriorityQueueWithCallbacksForSize(log, nil, 3)
	q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
	q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 4})
	q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 6})
	q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 5})
	e1, ok := q.Extract()
	//assert ok
	assert.True(t, ok)
	//assert extracted element is subaccountid1
	assert.Equal(t, subaccountid1, e1.SubaccountID)
	assert.Equal(t, int64(4), e1.ModifiedAt)

	e2, ok := q.Extract()
	assert.Equal(t, subaccountid3, e2.SubaccountID)
	assert.Equal(t, int64(6), e2.ModifiedAt)
	assert.Equal(t, "true", e2.BetaEnabled)
	assert.True(t, ok)

	//assert empty queue
	assert.True(t, q.IsEmpty())
}

func TestUpdateScenario2(t *testing.T) {
	q := NewPriorityQueueWithCallbacksForSize(log, nil, 3)
	q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "false", ModifiedAt: 3})
	q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 4})
	q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 6})
	q.Insert(QueueElement{SubaccountID: subaccountid1, BetaEnabled: "false", ModifiedAt: 5})
	e1, ok := q.Extract()
	//assert ok
	assert.True(t, ok)
	//assert extracted element is subaccountid1
	assert.Equal(t, subaccountid1, e1.SubaccountID)
	assert.Equal(t, int64(5), e1.ModifiedAt)
	assert.Equal(t, "false", e1.BetaEnabled)

	e2, ok := q.Extract()
	assert.Equal(t, subaccountid3, e2.SubaccountID)
	assert.Equal(t, int64(6), e2.ModifiedAt)
	assert.Equal(t, "true", e2.BetaEnabled)
	assert.True(t, ok)

	//assert empty queue
	assert.True(t, q.IsEmpty())
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
