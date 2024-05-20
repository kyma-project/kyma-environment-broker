package postsql

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
)

type instanceArchived struct {
	factory postsql.Factory
}

func NewInstanceArchived(sess postsql.Factory) *instanceArchived {
	return &instanceArchived{
		factory: sess,
	}
}

func (s *instanceArchived) GetByInstanceID(instanceId string) (internal.InstanceArchived, error) {
	dto, err := s.factory.NewReadSession().GetInstanceArchivedByID(instanceId)
	return dbmodel.NewInstanceArchivedFromDTO(dto), err
}

func (s *instanceArchived) Insert(instance internal.InstanceArchived) error {
	return s.factory.NewWriteSession().InsertInstanceArchived(dbmodel.NewInstanceArchivedDTO(instance))
}

func (s *instanceArchived) TotalNumberOfInstancesArchived() (int, error) {
	return s.factory.NewReadSession().TotalNumberOfInstancesArchived()
}

func (s *instanceArchived) TotalNumberOfInstancesArchivedForGlobalAccountID(globalAccountID string, planID string) (int, error) {
	return s.factory.NewReadSession().TotalNumberOfInstancesArchivedForGlobalAccountID(globalAccountID, planID)
}
