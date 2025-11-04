package deprovisioning

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/archive"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
)

const (
	dbRetryBackoff = 2 * time.Second
)

type ArchivingStep struct {
	operations        storage.Operations
	instances         storage.Instances
	instancesArchived storage.InstancesArchived
}

func NewArchivingStep(db storage.BrokerStorage) *ArchivingStep {
	return &ArchivingStep{
		operations:        db.Operations(),
		instances:         db.Instances(),
		instancesArchived: db.InstancesArchived(),
	}
}

func (s *ArchivingStep) Name() string {
	return "Archiving"
}

func (s *ArchivingStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.Temporary {
		logger.Info("suspension operation, skipping")
		return operation, 0, nil
	}
	if operation.ExcutedButNotCompleted != nil && len(operation.ExcutedButNotCompleted) > 0 {
		logger.Info("operation needs a retrigger, skipping")
		return operation, 0, nil
	}

	// check if the archived instance already exists
	_, err := s.instancesArchived.GetByInstanceID(operation.InstanceID)
	switch {
	case err == nil:
		logger.Warn("archived instance already existis, skipping")
		return operation, 0, nil
	case !dberr.IsNotFound(err):
		logger.Error("Unable to get archived instance")
		return operation, dbRetryBackoff, nil
	}

	instance, err := s.instances.GetByID(operation.InstanceID)
	switch {
	case dberr.IsNotFound(err):
		logger.Error("Instance not found")
	case err != nil:
		return operation, dbRetryBackoff, nil
	}

	operations, err := s.operations.ListOperationsByInstanceID(operation.InstanceID)
	if err != nil {
		logger.Error("unable to get operations for given instance")
		return operation, dbRetryBackoff, nil
	}

	var archived internal.InstanceArchived
	if instance != nil {
		logger.Info(fmt.Sprintf("Creating instance archived from the instance and %d operations", len(operations)))
		archived, err = archive.NewInstanceArchivedFromOperationsAndInstance(*instance, operations)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to create instance archived: %s", err.Error()))
			return operation, 0, nil
		}
	} else {
		logger.Info(fmt.Sprintf("Creating instance archived from %d operations", len(operations)))
		archived, err = archive.NewInstanceArchivedFromOperations(operations)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to create instance archived: %s", err.Error()))
			return operation, 0, nil
		}
	}

	err = s.instancesArchived.Insert(archived)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to insert archived instance: %s", err.Error()))
		return operation, dbRetryBackoff, nil
	}

	return operation, 0, nil
}
