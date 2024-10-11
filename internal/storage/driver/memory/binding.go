package memory

import (
	"sync"

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

	if _, found := s.data[binding.ID]; found {
		return dberr.AlreadyExists("binding with id %s already exist", binding.ID)
	}
	s.data[binding.ID] = *binding

	return nil
}

func (s *Binding) DeleteByBindingID(bindingID string) error {
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
