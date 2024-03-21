package archive

import (
	"fmt"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"k8s.io/apimachinery/pkg/api/errors"
)

type Service struct {
	instances     storage.Instances
	operations    storage.Operations
	runtimeStates storage.RuntimeStates
	archived      storage.InstancesArchived

	dryRun          bool
	performDeletion bool
	batchSize       int
}

func NewService(db storage.BrokerStorage, dryRun bool, performDeletion bool, batchSize int) *Service {
	return &Service{
		instances:       db.Instances(),
		operations:      db.Operations(),
		runtimeStates:   db.RuntimeStates(),
		archived:        db.InstancesArchived(),
		dryRun:          dryRun,
		performDeletion: performDeletion,
		batchSize:       batchSize,
	}
}

func (s *Service) Run() (error, int, int) {
	l := slog.Default()

	instanceIDs, err := s.instances.ListDeletedInstanceIDs(s.batchSize)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to get instance IDs: %s", err.Error()))
		return err, 0, 0
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
			return fmt.Errorf("instance exists"), numberOfInstancesProcessed, numberOfOperationsDeleted
		}
		if !dberr.IsNotFound(errInstance) {
			return errInstance, numberOfInstancesProcessed, numberOfOperationsDeleted
		}

		operations, err := s.operations.ListOperationsByInstanceID(instanceId)
		if err != nil {
			return err, numberOfInstancesProcessed, numberOfOperationsDeleted
		}

		archived, err := NewInstanceArchivedFromOperations(operations)
		if err != nil {
			return err, numberOfInstancesProcessed, numberOfOperationsDeleted
		}

		if s.dryRun {
			logger.Debug(fmt.Sprintf("DryRun: Instance would be archived: %+v", archived))
		} else {
			logger.Debug("Archiving the instance")
			// do not throw error if the instance is already archived
			err = s.archived.Insert(archived)
			if err != nil && !errors.IsAlreadyExists(err) {
				return err, numberOfInstancesProcessed, numberOfOperationsDeleted
			}
		}

		for _, operation := range operations {
			log := logger.With("operationID", operation.ID).With("type", operation.Type)

			if s.dryRun {
				log.Debug("DryRun: Operation would be deleted")
				continue
			}

			// first - delete all runtime states
			// second - delete the operation
			// If the deletion of operation fails, it can be retried, because such instance ID will be fetched by
			// the next run of ListDeletedInstanceIDs() method.

			log.Debug("Deleting runtime states for operation")
			err := s.runtimeStates.DeleteByOperationID(operation.ID)
			if err != nil {
				return err, numberOfInstancesProcessed, numberOfOperationsDeleted
			}
			log.Debug("Deleting operation")
			err = s.operations.DeleteByID(operation.ID)
			if err != nil {
				return err, numberOfInstancesProcessed, numberOfOperationsDeleted
			}
			numberOfOperationsDeleted++
		}
		numberOfInstancesProcessed++
	}

	return nil, numberOfInstancesProcessed, numberOfOperationsDeleted
}
