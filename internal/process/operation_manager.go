package process

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	kebErr "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type OperationManager struct {
	storage     storage.Operations
	dependecies []kebErr.ErrComponent
}

func NewOperationManager(storage storage.Operations, dependecies ...kebErr.ErrComponent) *OperationManager {
	return &OperationManager{storage: storage, dependecies: dependecies}
}

// OperationSucceeded marks the operation as succeeded and returns status of the operation's update
func (om *OperationManager) OperationSucceeded(operation internal.Operation, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.update(operation, domain.Succeeded, description, log)
}

// OperationFailed marks the operation as failed and returns status of the operation's update
func (om *OperationManager) OperationFailed(operation internal.Operation, description string, err error, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	operation.LastError = om.setLastError(err, description)
	op, t, _ := om.update(operation, domain.Failed, description, log)
	// repeat in case of storage error
	if t != 0 {
		return op, t, nil
	}

	var retErr error
	if err == nil {
		// no exact err passed in
		retErr = fmt.Errorf(description)
	} else {
		// keep the original err object for error categorizer
		retErr = fmt.Errorf("%s: %w", description, err)
	}

	log.Errorf("Step execution failed: %v", retErr)
	operation.EventErrorf(err, "operation failed")

	return op, 0, retErr
}

// OperationCanceled marks the operation as canceled and returns status of the operation's update
func (om *OperationManager) OperationCanceled(operation internal.Operation, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.update(operation, orchestration.Canceled, description, log)
}

// RetryOperation checks if operation should be retried or if it's the status should be marked as failed
func (om *OperationManager) RetryOperation(operation internal.Operation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	log.Infof("Retry Operation was triggered with message: %s", errorMessage)
	log.Infof("Retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if time.Since(operation.UpdatedAt) < maxTime {
		return operation, retryInterval, nil
	}
	log.Errorf("Aborting after %s of failing retries", maxTime.String())
	op, retry, err := om.OperationFailed(operation, errorMessage, err, log)
	if err == nil {
		err = fmt.Errorf("Too many retries")
	} else {
		err = fmt.Errorf("Failed to set status for operation after too many retries: %v", err)
	}
	return op, retry, err
}

// RetryOperationWithoutFail checks if operation should be retried or updates the status to InProgress, but omits setting the operation to failed if maxTime is reached
func (om *OperationManager) RetryOperationWithoutFail(operation internal.Operation, stepName string, description string, retryInterval, maxTime time.Duration, log logrus.FieldLogger, opErr error) (internal.Operation, time.Duration, error) {

	if opErr != nil {
		log.Warnf("error while invoking the step: %s", opErr.Error())
	}

	log.Infof("retrying for %s in %s steps", maxTime.String(), retryInterval.String())
	if time.Since(operation.UpdatedAt) < maxTime {
		return operation, retryInterval, nil
	}
	// update description to track failed steps
	op, repeat, err := om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.State = domain.InProgress
		operation.Description = description
		operation.ExcutedButNotCompleted = append(operation.ExcutedButNotCompleted, stepName)
	}, log)
	if repeat != 0 {
		return op, repeat, err
	}

	op.EventErrorf(fmt.Errorf(description), "step %s failed retries: operation continues", stepName)
	if opErr != nil {
		log.Errorf("omitting after %s of failing retries, last error: %s", maxTime.String(), opErr.Error())
	} else {
		log.Errorf("omitting after %s of failing retries", maxTime.String())
	}
	return op, 0, nil
}

// RetryOperationOnce retries the operation once and fails the operation when call second time
func (om *OperationManager) RetryOperationOnce(operation internal.Operation, errorMessage string, err error, wait time.Duration, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.RetryOperation(operation, errorMessage, err, wait, wait+1, log)
}

// UpdateOperation updates a given operation and handles conflict situation
func (om *OperationManager) UpdateOperation(operation internal.Operation, update func(operation *internal.Operation), log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	update(&operation)
	op, err := om.storage.UpdateOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err = om.storage.GetOperationByID(operation.ID)
			if err != nil {
				log.Errorf("while getting operation: %v", err)
				return operation, 1 * time.Minute, err
			}
			op.Merge(&operation)
			update(op)
			op, err = om.storage.UpdateOperation(*op)
			if err != nil {
				log.Errorf("while updating operation after conflict: %v", err)
				return operation, 1 * time.Minute, err
			}
		}
	case err != nil:
		log.Errorf("while updating operation: %v", err)
		return operation, 1 * time.Minute, err
	}

	return *op, 0, nil
}

func (om *OperationManager) MarkStepAsExcutedButNotCompleted(operation internal.Operation, stepName string, msg string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	op, repeat, err := om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.ExcutedButNotCompleted = append(operation.ExcutedButNotCompleted, stepName)
	}, log)
	if repeat != 0 {
		return op, repeat, err
	}

	op.EventErrorf(fmt.Errorf(msg), "step %s failed: operation continues", stepName)
	log.Errorf(msg)
	return op, 0, nil
}

func (om *OperationManager) update(operation internal.Operation, state domain.LastOperationState, description string, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	return om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.State = state
		operation.Description = description
	}, log)
}

func (om *OperationManager) setLastError(err error, description string) kebErr.LastError {
	toPersist := kebErr.LastErrorJSON{}

	if err == nil || err.Error() == "" {
		toPersist.Message = string(kebErr.ErrNotSet)
	} else {
		toPersist.Message = err.Error()
	}

	if description == "" {
		toPersist.Reason = kebErr.ErrMsgNotSet
	} else {
		toPersist.Reason = kebErr.ErrReason(description)
	}

	dependecies := om.dependecies
	if len(om.dependecies) == 0 {
		toPersist.Component = kebErr.ErrComponent(kebErr.ErrUnknown)
	} else {
		var sb strings.Builder
		for idx, dependency := range dependecies {
			sb.WriteString(string(dependency))
			if len(dependecies) > 1 && idx != len(dependecies)-1 {
				sb.WriteString(",")
			}
		}
		toPersist.Component = kebErr.ErrComponent(sb.String())
	}

	return toPersist.ToDTO()
}
