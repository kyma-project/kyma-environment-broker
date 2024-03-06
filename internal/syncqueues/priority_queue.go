package syncqueues

import (
	"fmt"
	"log/slog"
	"sync"
)

// Modified priority queue implementation based on heap
//  - priority based on minimal value of ModifiedAt
//  - semantics of insert operation is redefined
// 		we insert element with non-existing subaccountID,
// 		we update element with existing subaccountID if it is outdated

type SubaccountAwarePriorityQueue struct {
	elements []QueueElement
	idx      map[string]int
	size     int
	mutex    sync.Mutex
	log      *slog.Logger
}

func NewPriorityQueue(log *slog.Logger) *SubaccountAwarePriorityQueue {
	return &SubaccountAwarePriorityQueue{
		elements: make([]QueueElement, InitialQueueSize),
		idx:      make(map[string]int),
		size:     0,
		log:      log,
	}
}

func NewPriorityQueueForTests(log *slog.Logger, initialQueueSize int) *SubaccountAwarePriorityQueue {
	return &SubaccountAwarePriorityQueue{
		elements: make([]QueueElement, initialQueueSize),
		idx:      make(map[string]int),
		size:     0,
		log:      log,
	}
}

func (q *SubaccountAwarePriorityQueue) Insert(e QueueElement) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.size == cap(q.elements) {
		newElements := make([]QueueElement, q.size*2)
		copy(newElements, q.elements)
		q.elements = newElements
		q.log.Info(fmt.Sprintf("SimplePriorityQueue is full, resized to %v", q.size*2))
	}
	if idx, ok := q.idx[e.SubaccountID]; ok {
		if q.elements[idx].ModifiedAt > e.ModifiedAt {
			// event is outdated, do not insert
			return
		}
		// extraction and re-insertion
		q.elements[idx].ModifiedAt = -1
		q.siftUpFrom(idx)
		// updated element is at the top of the heap
		// remove it
		q.elements[0] = q.elements[q.size-1]
		q.size--
		q.siftDown()
	} else {
		q.idx[e.SubaccountID] = q.size
	}
	q.elements[q.size] = e
	q.size++
	q.siftUp()
	q.dumpQueue()
}

func (q *SubaccountAwarePriorityQueue) dumpQueue() {
	for i := 0; i < q.size; i++ {
		q.log.Info(fmt.Sprintf("Element: %v", q.elements[i]))
	}
	q.log.Info(fmt.Sprintf("Map: %v", q.idx))
}

func (q *SubaccountAwarePriorityQueue) Extract() QueueElement {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.size == 0 {
		q.log.Error("SimplePriorityQueue is empty, cannot extract element")
		return QueueElement{}
	}
	e := q.elements[0]
	q.swap(0, q.size-1)
	q.size--
	delete(q.idx, e.SubaccountID)
	q.siftDown()
	return e
}

func (q *SubaccountAwarePriorityQueue) IsEmpty() bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.size == 0
}

func (q *SubaccountAwarePriorityQueue) siftUp() {
	i := q.size - 1
	q.siftUpFrom(i)
}

func (q *SubaccountAwarePriorityQueue) siftUpFrom(i int) {
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

func (q *SubaccountAwarePriorityQueue) swap(i int, parent int) {
	q.elements[i],
		q.elements[parent],
		q.idx[q.elements[i].SubaccountID],
		q.idx[q.elements[parent].SubaccountID] = q.elements[parent], q.elements[i], q.idx[q.elements[parent].SubaccountID], q.idx[q.elements[i].SubaccountID]
}

func (q *SubaccountAwarePriorityQueue) siftDown() {
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
