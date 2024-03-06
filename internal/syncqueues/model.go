package syncqueues

const (
	// MaxQueueSize is the maximum size of the queue
	MaxQueueSize = 2048
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
