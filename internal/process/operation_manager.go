package process

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebErr "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type OperationManager struct {
	storage   storage.Operations
	component kebErr.Component
	step      string

	// stores timestamp to calculate timeout in retry* methods, the key is the operation.ID
	retryTimestamps map[string]time.Time
	mu              sync.RWMutex
}

func NewOperationManager(storage storage.Operations, step string, component kebErr.Component) *OperationManager {
	op := &OperationManager{storage: storage, component: component, step: step, retryTimestamps: make(map[string]time.Time)}
	go func(op *OperationManager, step string) {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			runTimestampGC(op, step)
		}
	}(op, step)
	return op
}

func runTimestampGC(op *OperationManager, step string) {
	numberOfDeletions := 0
	op.mu.Lock()
	for opId, ts := range op.retryTimestamps {
		if time.Since(ts) > 48*time.Hour {
			delete(op.retryTimestamps, opId)
			numberOfDeletions++
		}
	}
	op.mu.Unlock()
	if numberOfDeletions > 0 {
		slog.Info("Operation Manager for step %s has deleted %d old timestamps", step, numberOfDeletions)
	}
}

// OperationSucceeded marks the operation as succeeded and returns status of the operation's update
func (om *OperationManager) OperationSucceeded(operation internal.Operation, description string, log *slog.Logger) (internal.Operation, time.Duration, error) {
	return om.update(operation, domain.Succeeded, description, log)
}

// OperationFailed marks the operation as failed and returns status of the operation's update
func (om *OperationManager) OperationFailed(operation internal.Operation, description string, err error, log *slog.Logger) (internal.Operation, time.Duration, error) {
	operation.LastError = kebErr.LastError{
		Reason:    kebErr.Reason(description),
		Component: om.component,
		Step:      om.step,
	}
	if err != nil {
		operation.LastError.Message = err.Error()
	}

	op, t, _ := om.update(operation, domain.Failed, description, log)
	// repeat in case of storage error
	if t != 0 {
		return op, t, nil
	}

	var retErr error
	if err == nil {
		// no exact err passed in
		retErr = fmt.Errorf("%s", description)
	} else {
		// keep the original err object for error categorizer
		retErr = fmt.Errorf("%s: %w", description, err)
	}

	log.Error(fmt.Sprintf("Step execution failed: %v", retErr))
	operation.EventErrorf(retErr, "operation failed")

	return op, 0, retErr
}

// OperationCanceled marks the operation as canceled and returns status of the operation's update
func (om *OperationManager) OperationCanceled(operation internal.Operation, description string, log *slog.Logger) (internal.Operation, time.Duration, error) {
	return om.update(operation, internal.OperationStateCanceled, description, log)
}

// RetryOperation checks if operation should be retried or if it's the status should be marked as failed
func (om *OperationManager) RetryOperation(operation internal.Operation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log *slog.Logger) (internal.Operation, time.Duration, error) {
	log.Info(fmt.Sprintf("Retry Operation was called with message: %s", errorMessage))

	log.Debug(fmt.Sprintf("Retry Operation map size is: %d", len(om.retryTimestamps)))
	om.storeTimestampIfMissing(operation.ID)
	if !om.isTimeoutOccurred(operation.ID, maxTime) {
		remainingTime := om.getRemainingTime(operation.ID, maxTime)
		log.Info(fmt.Sprintf("Retrying for %s in %s intervals %d minutes left", maxTime.String(), retryInterval.String(), int(remainingTime.Round(time.Second).Minutes())))
		return operation, retryInterval, nil
	}

	log.Error(fmt.Sprintf("Failing operation after %s of failing retries", maxTime.String()))
	op, retry, err := om.OperationFailed(operation, errorMessage, err, log)
	if err == nil {
		err = fmt.Errorf("too many retries")
	} else {
		err = fmt.Errorf("failed to set status `Failed` for operation after too many retries: %v", err)
	}
	return op, retry, err
}

func (om *OperationManager) RetryOperationWithCreatedAt(operation internal.Operation, errorMessage string, err error, retryInterval time.Duration, maxTime time.Duration, log *slog.Logger) (internal.Operation, time.Duration, error) {
	log.Info(fmt.Sprintf("Retry Operation was called with message: %s", errorMessage))

	om.storeCreatedAtIfMissing(operation.ID, operation.CreatedAt)
	if !om.isTimeoutOccurred(operation.ID, maxTime) {
		remainingTime := om.getRemainingTime(operation.ID, maxTime)
		log.Info(fmt.Sprintf("Retrying for %s in %s intervals %d minutes left", maxTime.String(), retryInterval.String(), int(remainingTime.Round(time.Second).Minutes())))
		return operation, retryInterval, nil
	}

	log.Error(fmt.Sprintf("Failing operation after %s of failing retries", maxTime.String()))
	op, retry, err := om.OperationFailed(operation, errorMessage, err, log)
	if err == nil {
		err = fmt.Errorf("too many retries")
	} else {
		err = fmt.Errorf("failed to set status `Failed` for operation after too many retries: %v", err)
	}
	return op, retry, err
}

// RetryOperationWithoutFail checks if operation should be retried or updates the status to InProgress, but omits setting the operation to failed if maxTime is reached
func (om *OperationManager) RetryOperationWithoutFail(operation internal.Operation, stepName string, description string, retryInterval, maxTime time.Duration, log *slog.Logger, opErr error) (internal.Operation, time.Duration, error) {

	if opErr != nil {
		log.Warn(fmt.Sprintf("error while invoking the step: %s", opErr.Error()))
	}

	log.Info(fmt.Sprintf("retrying for %s in %s steps", maxTime.String(), retryInterval.String()))
	om.storeTimestampIfMissing(operation.ID)
	if !om.isTimeoutOccurred(operation.ID, maxTime) {
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

	op.EventErrorf(fmt.Errorf("%s", description), "step %s failed all retries: operation continues", stepName)
	if opErr != nil {
		log.Error(fmt.Sprintf("quiting step after %s of failing retries, last error: %s", maxTime.String(), opErr.Error()))
	} else {
		log.Error(fmt.Sprintf("quiting step after %s of failing retries", maxTime.String()))
	}
	return op, 0, nil
}

// RetryOperationOnce retries the operation once and fails the operation when call second time
func (om *OperationManager) RetryOperationOnce(operation internal.Operation, errorMessage string, err error, wait time.Duration, log *slog.Logger) (internal.Operation, time.Duration, error) {
	return om.RetryOperation(operation, errorMessage, err, wait, wait+1, log)
}

// UpdateOperation updates a given operation and handles conflict situation
// The DB update call must be done even if there is no any change in the operation - this is required to update the operation's `UpdatedAt` field.
func (om *OperationManager) UpdateOperation(operation internal.Operation, update func(operation *internal.Operation), log *slog.Logger) (internal.Operation, time.Duration, error) {
	update(&operation)
	op, err := om.storage.UpdateOperation(operation)
	switch {
	case dberr.IsConflict(err):
		{
			op, err = om.storage.GetOperationByID(operation.ID)
			if err != nil {
				log.Error(fmt.Sprintf("while getting operation: %v", err))
				return operation, 1 * time.Minute, err
			}
			// do not optimize the flow by skipping the update call - it's required to update the `UpdatedAt` field
			op.Merge(&operation)
			update(op)
			op, err = om.storage.UpdateOperation(*op)
			if err != nil {
				log.Error(fmt.Sprintf("while updating operation after conflict: %v", err))
				return operation, 1 * time.Minute, err
			}
		}
	case err != nil:
		log.Error(fmt.Sprintf("while updating operation: %v", err))
		return operation, 1 * time.Minute, err
	}

	return *op, 0, nil
}

func (om *OperationManager) MarkStepAsExecutedButNotCompleted(operation internal.Operation, stepName string, msg string, log *slog.Logger) (internal.Operation, time.Duration, error) {
	op, repeat, err := om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.ExcutedButNotCompleted = append(operation.ExcutedButNotCompleted, stepName)
	}, log)
	if repeat != 0 {
		return op, repeat, err
	}

	op.EventErrorf(fmt.Errorf("%s", msg), "step %s failed: operation continues", stepName)
	log.Error(msg)
	return op, 0, nil
}

func (om *OperationManager) update(operation internal.Operation, state domain.LastOperationState, description string, log *slog.Logger) (internal.Operation, time.Duration, error) {
	return om.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.State = state
		operation.Description = description
	}, log)
}

func (om *OperationManager) storeTimestampIfMissing(id string) {
	om.mu.Lock()
	defer om.mu.Unlock()
	if om.retryTimestamps[id].IsZero() {
		om.retryTimestamps[id] = time.Now()
	}
}

func (om *OperationManager) storeCreatedAtIfMissing(id string, createdAt time.Time) {
	om.mu.Lock()
	defer om.mu.Unlock()
	if om.retryTimestamps[id].IsZero() {
		om.retryTimestamps[id] = createdAt
	}
}

func (om *OperationManager) isTimeoutOccurred(id string, maxTime time.Duration) bool {
	om.mu.RLock()
	defer om.mu.RUnlock()
	since := time.Since(om.retryTimestamps[id])
	return !om.retryTimestamps[id].IsZero() && since > maxTime
}

func (om *OperationManager) getRemainingTime(id string, maxTime time.Duration) time.Duration {
	om.mu.RLock()
	defer om.mu.RUnlock()
	if om.retryTimestamps[id].IsZero() {
		return maxTime
	} else {
		remaining := maxTime - time.Since(om.retryTimestamps[id])
		if remaining < 0 {
			return 0
		}
		return remaining
	}
}
