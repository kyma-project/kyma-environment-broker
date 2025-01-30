package memory

import (
	"sort"
	"sync"

	"github.com/kyma-project/kyma-environment-broker/common/pagination"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/pivotal-cf/brokerapi/v12/domain"

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

func (s *InstanceArchivedInMemoryStorage) TotalNumberOfInstancesArchivedForGlobalAccountID(globalAccountID string, planID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	numberOfInstances := 0
	for _, inst := range s.data {
		if inst.GlobalAccountID == globalAccountID &&
			inst.PlanID == planID &&
			inst.ProvisioningState == domain.Succeeded {
			numberOfInstances++
		}
	}

	return numberOfInstances, nil
}

func (s *InstanceArchivedInMemoryStorage) List(filter dbmodel.InstanceFilter) ([]internal.InstanceArchived, int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var toReturn []internal.InstanceArchived

	instancesArchived := s.filterInstancesArchived(filter)
	sortInstancesArchivedByLastDeprovisioningFinishedAt(instancesArchived)

	offset := pagination.ConvertPageAndPageSizeToOffset(filter.PageSize, filter.Page)
	for i := offset; (filter.PageSize < 1 || i < offset+filter.PageSize) && i < len(instancesArchived); i++ {
		toReturn = append(toReturn, s.data[instancesArchived[i].InstanceID])
	}

	return toReturn, len(toReturn), len(instancesArchived), nil
}

func (s *InstanceArchivedInMemoryStorage) filterInstancesArchived(filter dbmodel.InstanceFilter) []internal.InstanceArchived {
	instancesArchived := make([]internal.InstanceArchived, 0, len(s.data))
	var ok bool
	equal := func(a, b string) bool {
		return a == b
	}

	for _, i := range s.data {
		if ok = matchFilter(i.InstanceID, filter.InstanceIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(i.GlobalAccountID, filter.GlobalAccountIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(i.SubaccountID, filter.SubAccountIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(i.PlanName, filter.Plans, equal); !ok {
			continue
		}
		if ok = matchFilter(i.Region, filter.Regions, equal); !ok {
			continue
		}
		if ok = matchFilter(i.LastRuntimeID, filter.RuntimeIDs, equal); !ok {
			continue
		}
		if ok = matchFilter(i.ShootName, filter.Shoots, equal); !ok {
			continue
		}

		instancesArchived = append(instancesArchived, i)
	}

	return instancesArchived
}

func sortInstancesArchivedByLastDeprovisioningFinishedAt(instances []internal.InstanceArchived) {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].LastDeprovisioningFinishedAt.Before(instances[j].LastDeprovisioningFinishedAt)
	})
}
