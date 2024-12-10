package provisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

// StartStep changes the state from pending to in progress if necessary
type StartStep struct {
	operationStorage storage.Operations
	instanceStorage  storage.Instances
	operationManager *process.OperationManager
}

func NewStartStep(os storage.Operations, is storage.Instances) *StartStep {
	step := &StartStep{
		operationStorage: os,
		instanceStorage:  is,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *StartStep) Name() string {
	return "Starting"
}

func (s *StartStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.State != orchestration.Pending {
		return operation, 0, nil
	}

	deprovisionOp, err := s.operationStorage.GetDeprovisioningOperationByInstanceID(operation.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		log.Error(fmt.Sprintf("Unable to get deprovisioning operation: %s", err.Error()))
		return operation, time.Second, nil
	}
	if deprovisionOp != nil && deprovisionOp.State == domain.InProgress {
		return operation, time.Minute, nil
	}

	// if there was a deprovisioning process before, take new InstanceDetails
	if deprovisionOp != nil {
		inst, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			if dberr.IsNotFound(err) {
				log.Error("Instance does not exists.")
				return s.operationManager.OperationFailed(operation, "The instance does not exists", err, log)
			}
			log.Error(fmt.Sprintf("Unable to get the instance: %s", err.Error()))
			return operation, time.Second, nil
		}
		log.Info("Setting the newest InstanceDetails")
		operation.InstanceDetails, err = inst.GetInstanceDetails()
		if err != nil {
			log.Error(fmt.Sprintf("Unable to provide Instance details: %s", err.Error()))
			return s.operationManager.OperationFailed(operation, "Unable to provide Instance details", err, log)
		}
	}
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		log.Warn(fmt.Sprintf("Failed to get last operation for ERSContext: %s", err.Error()))
		return operation, time.Minute, nil
	}
	log.Info("Setting the operation to 'InProgress'")
	newOp, retry, _ := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		if lastOp != nil {
			op.ProvisioningParameters.ErsContext = internal.InheritMissingERSContext(op.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
		}
		op.State = domain.InProgress
	}, log)
	operation = newOp
	if retry > 0 {
		return operation, time.Second, nil
	}

	return operation, 0, nil
}
