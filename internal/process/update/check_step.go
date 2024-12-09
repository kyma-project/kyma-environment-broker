package update

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

// CheckStep checks if the SKR is updated
type CheckStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.OperationManager
	provisioningTimeout time.Duration
}

func NewCheckStep(os storage.Operations,
	provisionerClient provisioner.Client,
	provisioningTimeout time.Duration) *CheckStep {
	step := &CheckStep{
		provisionerClient:   provisionerClient,
		provisioningTimeout: provisioningTimeout,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.ProvisionerDependency)
	return step
}

var _ process.Step = (*CheckStep)(nil)

func (s *CheckStep) Name() string {
	return "Check_Runtime"
}

func (s *CheckStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *CheckStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		log.Error("Runtime ID is empty")
		return s.operationManager.OperationFailed(operation, "Runtime ID is empty", nil, log)
	}
	return s.checkRuntimeStatus(operation, log.With("runtimeID", operation.RuntimeID))
}

func (s *CheckStep) checkRuntimeStatus(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.ProvisionerOperationID == "" {
		// it can happen, when only KIM is involved in the process
		log.Info("Provisioner operation ID is empty, skipping")
		return operation, 0, nil
	}

	if time.Since(operation.UpdatedAt) > s.provisioningTimeout {
		log.Info(fmt.Sprintf("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt))
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", s.provisioningTimeout), nil, log)
	}

	status, err := s.provisionerClient.RuntimeOperationStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.ProvisionerOperationID)
	if err != nil {
		log.Error(fmt.Sprintf("call to provisioner RuntimeOperationStatus failed: %s", err.Error()))
		return operation, 1 * time.Minute, nil
	}
	log.Info(fmt.Sprintf("call to provisioner returned %s status", status.State.String()))

	var msg string
	if status.Message != nil {
		msg = *status.Message
	}

	switch status.State {
	case gqlschema.OperationStateSucceeded:
		return operation, 0, nil
	case gqlschema.OperationStateInProgress:
		return operation, time.Minute, nil
	case gqlschema.OperationStatePending:
		return operation, time.Minute, nil
	case gqlschema.OperationStateFailed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("provisioner client returns failed status: %s", msg), nil, log)
	}

	return s.operationManager.OperationFailed(operation, fmt.Sprintf("unsupported provisioner client status: %s", status.State.String()), nil, log)
}
