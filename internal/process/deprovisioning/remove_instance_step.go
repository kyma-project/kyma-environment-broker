package deprovisioning

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type RemoveInstanceStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	operationStorage storage.Operations
}

var _ process.Step = &RemoveInstanceStep{}

func NewRemoveInstanceStep(db storage.BrokerStorage) *RemoveInstanceStep {
	step := &RemoveInstanceStep{
		instanceStorage:  db.Instances(),
		operationStorage: db.Operations(),
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.KEBDependency)
	return step
}

func (s *RemoveInstanceStep) Name() string {
	return "Remove_Instance"
}

func (s *RemoveInstanceStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	var backoff time.Duration

	_, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		log.Info(fmt.Sprintf("instance already deleted: %v", err))
		return operation, 0 * time.Second, nil
	default:
		log.Error(fmt.Sprintf("unable to get instance from the storage: %v", err))
		return operation, 1 * time.Second, nil
	}

	if operation.Temporary {
		log.Info("Removing the RuntimeID field from the instance")
		backoff = s.removeRuntimeIDFromInstance(operation.InstanceID, log)
		if backoff != 0 {
			return operation, backoff, nil
		}

		log.Info("Removing the RuntimeID field from the operation")
		operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.RuntimeID = ""
		}, log)
	} else if operation.ExcutedButNotCompleted != nil {
		log.Info(fmt.Sprintf("Marking the instance needs to retry some steps (%s)", strings.Join(operation.ExcutedButNotCompleted, ", ")))
		backoff = s.markInstanceNeedsRetrySomeSteps(operation.InstanceID, log)
		if backoff != 0 {
			return operation, backoff, nil
		}
	} else {
		log.Info("Removing the instance permanently")

		// todo: the instance is needed in the next step
		backoff = s.removeInstancePermanently(operation.InstanceID, log)
		if backoff != 0 {
			return operation, backoff, nil
		}

		log.Info("Removing the userID field from the operation")
		operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.ProvisioningParameters.ErsContext.UserID = ""
		}, log)
	}

	return operation, backoff, nil
}

func (s *RemoveInstanceStep) removeRuntimeIDFromInstance(instanceID string, log *slog.Logger) time.Duration {
	backoff := time.Second

	instance, err := s.instanceStorage.GetByID(instanceID)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get instance %s from the storage: %s", instanceID, err))
		return backoff
	}

	// empty RuntimeID means there is no runtime in the Provisioner Domain
	instance.RuntimeID = ""
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		log.Error(fmt.Sprintf("unable to update instance %s in the storage: %s", instanceID, err))
		return backoff
	}

	return 0
}

func (s *RemoveInstanceStep) removeInstancePermanently(instanceID string, log *slog.Logger) time.Duration {
	err := s.instanceStorage.Delete(instanceID)
	if err != nil {
		log.Error(fmt.Sprintf("unable to remove instance %s from the storage: %s", instanceID, err))
		return 10 * time.Second
	}

	return 0
}

func (s *RemoveInstanceStep) markInstanceNeedsRetrySomeSteps(instanceID string, log *slog.Logger) time.Duration {
	backoff := time.Second

	instance, err := s.instanceStorage.GetByID(instanceID)
	if dberr.IsNotFound(err) {
		log.Warn(fmt.Sprintf("instance %s not found", instanceID))
		return 0
	}
	if err != nil {
		log.Error(fmt.Sprintf("unable to get instance %s from the storage: %s", instanceID, err))
		return backoff
	}

	instance.DeletedAt = time.Now()
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		log.Error(fmt.Sprintf("unable to update instance %s in the storage: %s", instanceID, err))
		return backoff
	}

	return 0
}
