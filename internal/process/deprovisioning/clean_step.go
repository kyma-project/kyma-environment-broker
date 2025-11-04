package deprovisioning

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type CleanStep struct {
	operations storage.Operations
}

func NewCleanStep(db storage.BrokerStorage) *CleanStep {
	return &CleanStep{
		operations: db.Operations(),
	}
}

func (s *CleanStep) Name() string {
	return "Clean"
}

func (s *CleanStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.Temporary {
		log.Info("suspension operation must not clean data")
		return operation, 0, nil
	}
	if operation.ExcutedButNotCompleted != nil && len(operation.ExcutedButNotCompleted) > 0 {
		log.Info("There are steps, which needs retry, skipping")
		return operation, 0, nil
	}

	operations, err := s.operations.ListOperationsByInstanceID(operation.InstanceID)
	if err != nil {
		return operation, dbRetryBackoff, nil
	}
	for _, op := range operations {
		log.Info(fmt.Sprintf("Removing operation %s", op.ID))
		err := s.operations.DeleteByID(op.ID)
		if err != nil {
			log.Error(fmt.Sprintf("unable to delete operation %s: %s", op.ID, err.Error()))
			return operation, dbRetryBackoff, nil
		}
	}
	log.Info(fmt.Sprintf("All runtime states and operations for the instance %s has been completely deleted!", operation.InstanceID))

	return operation, 0, nil
}
