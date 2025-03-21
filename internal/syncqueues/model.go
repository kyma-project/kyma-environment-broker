package syncqueues

const (
	DefaultQueueSize = 2048
)

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
	SubaccountID      string
	BetaEnabled       string
	UsedForProduction string
	ModifiedAt        int64
}

type EventHandler struct {
	OnInsert  func(queueSize int)
	OnExtract func(queueSize int, timeEnqueued int64)
}
