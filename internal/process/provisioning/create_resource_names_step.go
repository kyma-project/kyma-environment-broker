package provisioning

import (
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type CreateResourceNamesStep struct {
	operationManager *process.OperationManager
}

func NewCreateResourceNamesStep(os storage.Operations) *CreateResourceNamesStep {
	step := &CreateResourceNamesStep{}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *CreateResourceNamesStep) Name() string {
	return "Create_Resource_Names"
}

// The runtimeID could be generated and set in two different steps so we separated the logic to generate the Kyma name in this step
func (s *CreateResourceNamesStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		return s.operationManager.OperationFailed(operation, "RuntimeID not set, cannot create Kyma resource name and Runtime resource name", nil, log)
	}
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceName = steps.CreateKymaNameFromOperation(operation)
		op.RuntimeResourceName = steps.KymaRuntimeResourceName(operation)
	}, log)
}
