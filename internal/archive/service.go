package archive

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"k8s.io/apimachinery/pkg/api/errors"
)

type Service struct {
	instances  storage.Instances
	operations storage.Operations
	archived   storage.InstancesArchived

	dryRun          bool
	performDeletion bool
	batchSize       int
}

func NewService(db storage.BrokerStorage, dryRun bool, performDeletion bool, batchSize int) *Service {
	return &Service{
		instances:       db.Instances(),
		operations:      db.Operations(),
		archived:        db.InstancesArchived(),
		dryRun:          dryRun,
		performDeletion: performDeletion,
		batchSize:       batchSize,
	}
}

func (s *Service) Run() (int, int, error) {
	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	instanceIDs, err := s.instances.ListDeletedInstanceIDs(s.batchSize)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to get instance IDs: %s", err.Error()))
		return 0, 0, err
	}
	l.Info(fmt.Sprintf("Got %d instance IDs to process", len(instanceIDs)))

	numberOfInstancesProcessed := 0
	numberOfOperationsDeleted := 0

	for _, instanceId := range instanceIDs {
		logger := l.With("instanceID", instanceId)
		// check if the instance really does not exists
		instance, errInstance := s.instances.GetByID(instanceId)
		if errInstance == nil {
			logger.Error(fmt.Sprintf("the instance (createdAt: %s, planName: %s) still exists, aborting the process",
				instance.InstanceID, instance.CreatedAt))
			return numberOfInstancesProcessed, numberOfOperationsDeleted, fmt.Errorf("instance exists")
		}
		if !dberr.IsNotFound(errInstance) {
			return numberOfInstancesProcessed, numberOfOperationsDeleted, errInstance
		}

		operations, err := s.operations.ListOperationsByInstanceID(instanceId)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to get operations for instance: %s", err.Error()))
			continue
		}

		archived, err := NewInstanceArchivedFromOperations(operations)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to create archived instance: %s", err.Error()))
			continue
		}

		if s.dryRun {
			logger.Debug(fmt.Sprintf("DryRun: Instance would be archived: %+v", archived))
		} else {
			logger.Debug("Archiving the instance")
			// do not throw error if the instance is already archived
			err = s.archived.Insert(archived)
			if err != nil && !errors.IsAlreadyExists(err) {
				logger.Warn(fmt.Sprintf("Unable to insert archived instance [%+v]: %s", archived, err.Error()))
				continue
			}
		}

		for _, operation := range operations {
			logger := logger.With("operationID", operation.ID).With("type", operation.Type)

			if s.dryRun {
				logger.Debug("DryRun: Operation would be deleted")
				continue
			}

			if !s.performDeletion {
				logger.Debug("PerformDeletion is disabled, skipping operation deletion")
				continue
			}

			// first - delete all runtime states
			// second - delete the operation
			// If the deletion of operation fails, it can be retried, because such instance ID will be fetched by
			// the next run of ListDeletedInstanceIDs() method.

			logger.Debug("Deleting operation")
			err = s.operations.DeleteByID(operation.ID)
			if err != nil {
				logger.Error(fmt.Sprintf("Unable to delete operation: %s", err.Error()))
				continue
			}
			numberOfOperationsDeleted++
		}
		numberOfInstancesProcessed++
	}

	return numberOfInstancesProcessed, numberOfOperationsDeleted, nil
}
