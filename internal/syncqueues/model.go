package syncqueues

const (
	// InitialQueueSize is the initial size of the queue
	InitialQueueSize = 2048
)

type PriorityQueue interface {
	Insert(QueueElement)
	Extract() QueueElement
	IsEmpty() bool
}

type QueueElement struct {
	SubaccountID string
	BetaEnabled  string
	ModifiedAt   int64
}
