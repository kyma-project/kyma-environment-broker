package deprovisioning

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
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type InitStep struct {
	operationManager *process.OperationManager
	operationTimeout time.Duration
	operationStorage storage.Operations
	instanceStorage  storage.Instances
}

const (
	opRetryInterval = 1 * time.Minute
	dbRetryInterval = 10 * time.Second
	dbRetryTimeout  = 1 * time.Minute
)

func NewInitStep(operations storage.Operations, instances storage.Instances, operationTimeout time.Duration) *InitStep {
	step := &InitStep{
		operationTimeout: operationTimeout,
		operationStorage: operations,
		instanceStorage:  instances,
	}
	step.operationManager = process.NewOperationManager(operations, step.Name(), kebError.KEBDependency)
	return step
}

func (s *InitStep) Name() string {
	return "Initialisation"
}

func (s *InitStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {

	if operation.State != orchestration.Pending {
		return operation, 0, nil
	}
	// Check concurrent operation
	lastOp, err := s.operationStorage.GetLastOperation(operation.InstanceID)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "getting last operation", err, dbRetryInterval, dbRetryTimeout, log)
	}
	if !lastOp.IsFinished() {
		log.Info(fmt.Sprintf("waiting for %s operation (%s) to be finished", lastOp.Type, lastOp.ID))
		return s.operationManager.RetryOperation(operation, "waiting for operation to be finished", err, opRetryInterval, s.operationTimeout, log)
	}

	// read the instance details (it could happen that created deprovisioning operation has outdated one)
	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		if dberr.IsNotFound(err) {
			log.Warn("the instance already deprovisioned")
			return s.operationManager.OperationFailed(operation, "the instance was already deprovisioned", err, log)
		}
		return s.operationManager.RetryOperation(operation, "getting instance by ID", err, dbRetryInterval, dbRetryTimeout, log)
	}

	log.Info("Setting lastOperation ID in the instance")
	err = s.instanceStorage.UpdateInstanceLastOperation(operation.InstanceID, operation.ID)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "error while updating last operation ID", err, dbRetryInterval, dbRetryTimeout, log)
	}

	log.Info("Setting state 'in progress' and refreshing instance details")
	opr, delay, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.State = domain.InProgress
		op.InstanceDetails = instance.InstanceDetails
		op.ProvisioningParameters.ErsContext = internal.InheritMissingERSContext(op.ProvisioningParameters.ErsContext, lastOp.ProvisioningParameters.ErsContext)
	}, log)
	if delay != 0 {
		log.Error("unable to update the operation (move to 'in progress'), retrying")
		return s.operationManager.RetryOperation(operation, "unable to update the operation", err, dbRetryInterval, dbRetryTimeout, log)
	}

	return opr, 0, nil
}
