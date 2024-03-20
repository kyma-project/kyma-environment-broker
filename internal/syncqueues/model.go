package syncqueues

const (
	DefaultQueueSize = 2048
)

type PriorityQueue interface {
	Insert(QueueElement)
	Extract() QueueElement
	IsEmpty() bool
}

type MultiConsumerPriorityQueue interface {
	Insert(QueueElement)
	Extract() (QueueElement, bool)
	IsEmpty() bool
}

type ElementWrapper struct {
	QueueElement
	entryTime int64
}

type QueueElement struct {
	SubaccountID string
	BetaEnabled  string
	ModifiedAt   int64
}

type EventHandler struct {
	OnInsert  func(queueSize int)
	OnExtract func(queueSize int, timeEnqueued int64)
}
