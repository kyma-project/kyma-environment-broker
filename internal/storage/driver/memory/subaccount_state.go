package memory

import (
	"sync"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type SubaccountStates struct {
	mutex sync.Mutex

	subaccountStates map[string]internal.SubaccountState
}

func NewSubaccountStates() *SubaccountStates {
	return &SubaccountStates{
		subaccountStates: make(map[string]internal.SubaccountState, 0),
	}
}

func (s *SubaccountStates) DeleteState(subaccountID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.subaccountStates, subaccountID)

	return nil
}

func (s *SubaccountStates) UpsertState(subaccountState internal.SubaccountState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.subaccountStates[subaccountState.ID] = subaccountState

	return nil
}

func (s *SubaccountStates) ListStates() ([]internal.SubaccountState, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var states []internal.SubaccountState

	//convert map to slice
	for _, state := range s.subaccountStates {
		states = append(states, state)
	}

	return states, nil
}
