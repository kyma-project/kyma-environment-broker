package process

type fakeQueue struct {
	operationIDs []string
}

func NewFakeQueue() *fakeQueue {
	return &fakeQueue{operationIDs: make([]string, 0)}
}

func (q *fakeQueue) Add(opID string) {
	q.operationIDs = append(q.operationIDs, opID)
}
