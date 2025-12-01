package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
)

type Binding struct {
	mu   sync.Mutex
	data map[string]internal.Binding
}

func NewBinding() *Binding {
	return &Binding{
		data: make(map[string]internal.Binding),
	}
}

func (s *Binding) GetByBindingID(bindingID string) (*internal.Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	binding, found := s.data[bindingID]
	if !found {
		return nil, dberr.NotFound("binding with id %s not exist", bindingID)
	}
	return &binding, nil
}

func (s *Binding) Insert(binding *internal.Binding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if foundBinding, found := s.data[binding.ID]; found && binding.InstanceID == foundBinding.InstanceID {
		return dberr.AlreadyExists("binding with id %s already exists", binding.ID)
	}
	s.data[binding.ID] = *binding

	return nil
}

func (s *Binding) Update(binding *internal.Binding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if foundBinding, found := s.data[binding.ID]; !(found && binding.InstanceID == foundBinding.InstanceID) {
		return dberr.AlreadyExists("binding with id %s does not exist", binding.ID)
	}
	s.data[binding.ID] = *binding

	return nil
}

func (s *Binding) Delete(instanceID, bindingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, bindingID)
	return nil
}

func (s *Binding) ListByInstanceID(instanceID string) ([]internal.Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var bindings []internal.Binding
	for _, binding := range s.data {
		if binding.InstanceID == instanceID {
			bindings = append(bindings, binding)
		}
	}

	return bindings, nil
}

func (s *Binding) Get(instanceID string, bindingID string) (*internal.Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	binding, ok := s.data[bindingID]
	if ok && binding.InstanceID == instanceID {
		return &binding, nil
	} else if !ok {
		return nil, dberr.NotFound("binding with id %s does not exist", bindingID)
	} else {
		return nil, dberr.NotFound("binding with id %s does not exist for given instance ID", bindingID)
	}
}

func (s *Binding) ListExpired() ([]internal.Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentTime := time.Now().UTC()
	var bindings []internal.Binding
	for _, binding := range s.data {
		if binding.ExpiresAt.Before(currentTime) {
			bindings = append(bindings, binding)
		}
	}

	return bindings, nil
}

func (s *Binding) GetStatistics() (internal.BindingStats, error) {
	return internal.BindingStats{}, fmt.Errorf("not implemented")
}
