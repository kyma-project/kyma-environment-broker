package syncqueues

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Modified priority queue implementation based on heap
//  - priority based on minimal value of ModifiedAt
//  - semantics of insert operation is redefined
// 		we insert element with non-existing subaccountID,
// 		we update element with existing subaccountID if it is outdated

type SubaccountAwarePriorityQueueWithCallbacks struct {
	elements     []ElementWrapper
	idx          map[string]int
	size         int
	mutex        sync.Mutex
	log          *slog.Logger
	eventHandler *EventHandler
}

func NewPriorityQueueWithCallbacks(log *slog.Logger, eventHandler *EventHandler) *SubaccountAwarePriorityQueueWithCallbacks {
	return NewPriorityQueueWithCallbacksForSize(log, eventHandler, DefaultQueueSize)
}

func NewPriorityQueueWithCallbacksForSize(log *slog.Logger, eventHandler *EventHandler, initialQueueSize int) *SubaccountAwarePriorityQueueWithCallbacks {
	return &SubaccountAwarePriorityQueueWithCallbacks{
		elements:     make([]ElementWrapper, initialQueueSize),
		idx:          make(map[string]int),
		size:         0,
		log:          log,
		eventHandler: eventHandler,
	}
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) Insert(element QueueElement) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	e := ElementWrapper{
		QueueElement: element,
		entryTime:    time.Now().UnixNano(),
	}

	if q.size == cap(q.elements) {
		newElements := make([]ElementWrapper, q.size*2)
		copy(newElements, q.elements)
		q.elements = newElements
		q.log.Debug(fmt.Sprintf("Queue is full, resized to %v", q.size*2))
	}
	if idx, ok := q.idx[e.SubaccountID]; ok {
		if q.elements[idx].ModifiedAt > e.ModifiedAt {
			// new element is older than one already in the queue, do not insert
			q.log.Debug(fmt.Sprintf("Element with subaccountID %s, betaEnabled %s, usedForProduction %s, modifiedAt %d is older than %d - ignoring", e.SubaccountID, e.BetaEnabled, e.UsedForProduction, e.ModifiedAt, q.elements[idx].ModifiedAt))
			return
		}
		// extraction and re-insertion
		q.elements[idx].ModifiedAt = -1
		q.siftUpFrom(idx)
		// updated element is at the top of the heap
		// remove it
		q.elements[0] = q.elements[q.size-1]
		q.idx[q.elements[0].SubaccountID] = 0
		q.size--
		q.siftDown()
		q.log.Debug(fmt.Sprintf("Element with subaccountID %s is updated with betaEnabled %s, usedForProduction %s, modifiedAt %d", e.SubaccountID, e.BetaEnabled, e.UsedForProduction, e.ModifiedAt))
	} else {
		q.log.Debug(fmt.Sprintf("Element with subaccountID %s is inserted with betaEnabled %s, usedForProduction %s, modifiedAt %d", e.SubaccountID, e.BetaEnabled, e.UsedForProduction, e.ModifiedAt))
		q.idx[e.SubaccountID] = q.size
	}

	// insert new element
	q.elements[q.size] = e
	q.idx[e.SubaccountID] = q.size
	q.size++
	q.siftUp()

	if q.eventHandler != nil && q.eventHandler.OnInsert != nil {
		q.eventHandler.OnInsert(q.size)
	}

	//integrity check
	if q.size != len(q.idx) {
		q.log.Warn(fmt.Sprintf("Queue size %d is different from index size %d", q.size, len(q.idx)))
	}
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) Extract() (QueueElement, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.size == 0 {
		return QueueElement{}, false
	}
	e := q.elements[0]
	q.swap(0, q.size-1)
	q.size--
	delete(q.idx, e.SubaccountID)
	q.siftDown()

	if q.eventHandler != nil && q.eventHandler.OnExtract != nil {
		q.eventHandler.OnExtract(q.size, time.Now().UnixNano()-e.entryTime)
	}
	q.log.Debug(fmt.Sprintf("Element dequeued - subaccountID: %s, betaEnabled %s, usedForProduction %s", e.SubaccountID, e.BetaEnabled, e.UsedForProduction))
	return e.QueueElement, true
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) IsEmpty() bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return q.size == 0
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) siftUp() {
	i := q.size - 1
	q.siftUpFrom(i)
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) siftUpFrom(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if q.elements[i].ModifiedAt < q.elements[parent].ModifiedAt {
			q.swap(i, parent)
			i = parent
		} else {
			break
		}
	}
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) swap(i int, parent int) {
	q.elements[i],
		q.elements[parent],
		q.idx[q.elements[i].SubaccountID],
		q.idx[q.elements[parent].SubaccountID] =
		q.elements[parent],
		q.elements[i],
		q.idx[q.elements[parent].SubaccountID],
		q.idx[q.elements[i].SubaccountID]
}

func (q *SubaccountAwarePriorityQueueWithCallbacks) siftDown() {
	i := 0
	for {
		left := 2*i + 1
		right := 2*i + 2
		minimal := i
		if left < q.size && q.elements[left].ModifiedAt < q.elements[minimal].ModifiedAt {
			minimal = left
		}
		if right < q.size && q.elements[right].ModifiedAt < q.elements[minimal].ModifiedAt {
			minimal = right
		}
		if minimal != i {
			q.swap(i, minimal)
			i = minimal
		} else {
			break
		}
	}
}
