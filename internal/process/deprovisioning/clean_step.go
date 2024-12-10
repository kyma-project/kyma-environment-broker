package deprovisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type CleanStep struct {
	operations    storage.Operations
	runtimeStates storage.RuntimeStates
	dryRun        bool
}

func NewCleanStep(operations storage.Operations, runtimeStates storage.RuntimeStates, dryRun bool) *CleanStep {
	return &CleanStep{
		operations:    operations,
		runtimeStates: runtimeStates,
		dryRun:        dryRun,
	}
}

func (s *CleanStep) Name() string {
	return "Clean"
}

func (s *CleanStep) Dependency() kebError.Component {
	return kebError.NotSet
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
		log.Info(fmt.Sprintf("removing runtime states for operation %s", op.ID))
		if s.dryRun {
			log.Info("dry run mode, skipping")
			continue
		}
		err := s.runtimeStates.DeleteByOperationID(op.ID)
		if err != nil {
			log.Error(fmt.Sprintf("unable to delete runtime states: %s", err.Error()))
			return operation, dbRetryBackoff, nil
		}
	}
	for _, op := range operations {
		log.Info(fmt.Sprintf("Removing operation %s", op.ID))
		if s.dryRun {
			log.Info("dry run mode, skipping")
			continue
		}
		err := s.operations.DeleteByID(op.ID)
		if err != nil {
			log.Error(fmt.Sprintf("unable to delete operation %s: %s", op.ID, err.Error()))
			return operation, dbRetryBackoff, nil
		}
	}
	if !s.dryRun {
		log.Info(fmt.Sprintf("All runtime states and operations for the instance %s has been completely deleted!", operation.InstanceID))
	}
	return operation, 0, nil
}
