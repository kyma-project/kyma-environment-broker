package deprovisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type RemoveRuntimeStep struct {
	operationManager   *process.OperationManager
	instanceStorage    storage.Instances
	provisionerClient  provisioner.Client
	provisionerTimeout time.Duration
}

func NewRemoveRuntimeStep(os storage.Operations, is storage.Instances, cli provisioner.Client, provisionerTimeout time.Duration) *RemoveRuntimeStep {
	step := &RemoveRuntimeStep{
		instanceStorage:    is,
		provisionerClient:  cli,
		provisionerTimeout: provisionerTimeout,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.ProvisionerDependency)
	return step
}

func (s *RemoveRuntimeStep) Name() string {
	return "Remove_Runtime"
}

func (s *RemoveRuntimeStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *RemoveRuntimeStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {

	if operation.KimDeprovisionsOnly != nil && *operation.KimDeprovisionsOnly {
		log.Info(fmt.Sprintf("Skipping the step because the runtime %s/%s is not controlled by the provisioner", operation.GetRuntimeResourceName(), operation.GetRuntimeResourceName()))
		return operation, 0, nil
	}

	if time.Since(operation.UpdatedAt) > s.provisionerTimeout {
		log.Info(fmt.Sprintf("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt))
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisionerTimeout), nil, log)
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		log.Error(fmt.Sprintf("instance already deleted: %s", err))
		return operation, 0 * time.Second, nil
	default:
		log.Error(fmt.Sprintf("unable to get instance from storage: %s", err))
		return operation, 1 * time.Second, nil
	}
	if instance.RuntimeID == "" || operation.ProvisioningParameters.PlanID == broker.OwnClusterPlanID {
		// happens when provisioning process failed and Create_Runtime step was never reached
		// It can also happen when the SKR is suspended (technically deprovisioned)
		log.Info(fmt.Sprintf("Runtime does not exist for instance id %q", operation.InstanceID))
		return operation, 0 * time.Second, nil
	}

	if operation.ProvisionerOperationID == "" {
		provisionerResponse, err := s.provisionerClient.DeprovisionRuntime(instance.GlobalAccountID, instance.RuntimeID)
		if err != nil {
			log.Warn(fmt.Sprintf("unable to deprovision runtime: %s", err))
			return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), "unable to deprovision Runtime in Provisioner", 15*time.Second, 20*time.Minute, log, err)
		}
		log.Info(fmt.Sprintf("fetched ProvisionerOperationID=%s", provisionerResponse))
		repeat := time.Duration(0)
		operation, repeat, _ = s.operationManager.UpdateOperation(operation, func(o *internal.Operation) {
			o.ProvisionerOperationID = provisionerResponse
		}, log)
		if repeat != 0 {
			return operation, 5 * time.Second, nil
		}
	}

	log.Info("runtime deletion process initiated successfully")
	return operation, 0, nil
}
