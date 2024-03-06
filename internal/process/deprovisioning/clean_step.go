package deprovisioning

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
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

func (s *CleanStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.Temporary {
		log.Infof("suspension operation must not clean data")
		return operation, 0, nil
	}
	if operation.ExcutedButNotCompleted != nil && len(operation.ExcutedButNotCompleted) > 0 {
		log.Infof("There are steps, which needs retry, skipping")
		return operation, 0, nil
	}

	operations, err := s.operations.ListOperationsByInstanceID(operation.InstanceID)
	if err != nil {
		return operation, dbRetryBackoff, nil
	}
	for _, op := range operations {
		log.Infof("removing runtime states for operation %s", op.ID)
		if s.dryRun {
			log.Infof("dry run mode, skipping")
			continue
		}
		err := s.runtimeStates.DeleteByOperationID(op.ID)
		if err != nil {
			log.Errorf("unable to delete runtime states: %s", err.Error())
			return operation, dbRetryBackoff, nil
		}
	}
	for _, op := range operations {
		log.Infof("Removing operation %s", op.ID)
		if s.dryRun {
			log.Infof("dry run mode, skipping")
			continue
		}
		err := s.operations.DeleteByID(op.ID)
		if err != nil {
			log.Errorf("unable to delete operation %s: %s", op.ID, err.Error())
			return operation, dbRetryBackoff, nil
		}
	}
	if !s.dryRun {
		log.Infof("All runtime states and operations for the instance %s has been completely deleted!", operation.InstanceID)
	}
	return operation, 0, nil
}
