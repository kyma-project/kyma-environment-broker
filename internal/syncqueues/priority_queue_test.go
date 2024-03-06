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
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func TestQueue_Insert(t *testing.T) {

	q := NewPriorityQueue(log)
	element1 := QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 0}
	element2 := QueueElement{SubaccountID: subaccountid2, BetaEnabled: "false", ModifiedAt: 1}
	element3 := QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 2}
	element4 := QueueElement{SubaccountID: subaccountid1, BetaEnabled: "true", ModifiedAt: 3}
	t.Run("should insert element", func(t *testing.T) {
		q.Insert(element1)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert second element", func(t *testing.T) {
		q.Insert(element2)
		assert.Equal(t, 2, q.size)
	})
	t.Run("should extract element with minimal value of ModifiedAt", func(t *testing.T) {
		e := q.Extract()
		assert.Equal(t, subaccountid1, e.SubaccountID)
		assert.Equal(t, 1, q.size)
	})
	t.Run("should insert third element", func(t *testing.T) {
		q.Insert(element3)
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
	t.Run("should insert element ", func(t *testing.T) {
		q.Insert(element4)
		assert.Equal(t, 2, q.size)
	})
	t.Run("should update element again", func(t *testing.T) {
		q.Insert(QueueElement{SubaccountID: subaccountid3, BetaEnabled: "true", ModifiedAt: 5})
		assert.Equal(t, 2, q.size)
	})

}
