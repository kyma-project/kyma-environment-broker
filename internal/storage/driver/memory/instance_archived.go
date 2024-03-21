package memory

import (
	"sync"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
)

type InstanceArchivedInMemoryStorage struct {
	data map[string]internal.InstanceArchived

	mu sync.Mutex
}

func NewInstanceArchivedInMemoryStorage() *InstanceArchivedInMemoryStorage {
	return &InstanceArchivedInMemoryStorage{
		data: map[string]internal.InstanceArchived{},
		mu:   sync.Mutex{},
	}
}

func (s *InstanceArchivedInMemoryStorage) GetByInstanceID(instanceId string) (internal.InstanceArchived, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instance, found := s.data[instanceId]
	if !found {
		return instance, dberr.NotFound("instance archived not found")
	}
	return instance, nil
}

func (s *InstanceArchivedInMemoryStorage) Insert(instance internal.InstanceArchived) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[instance.InstanceID] = instance
	return nil
}

func (s *InstanceArchivedInMemoryStorage) TotalNumberOfInstancesArchived() (int, error) {
	return len(s.data), nil
}
