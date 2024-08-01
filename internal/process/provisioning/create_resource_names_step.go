package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type CreateResourceNamesStep struct {
	operationManager *process.OperationManager
}

func NewCreateResourceNamesStep(os storage.Operations) *CreateResourceNamesStep {
	return &CreateResourceNamesStep{
		operationManager: process.NewOperationManager(os),
	}
}

func (s *CreateResourceNamesStep) Name() string {
	return "Create_Resource_Names"
}

// The runtimeID could be generated and set in two different steps so we separated the logic to generate the Kyma name in this step
func (s *CreateResourceNamesStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		return s.operationManager.OperationFailed(operation, fmt.Sprint("RuntimeID not set, cannot create Kyma resource name and Runtime resource name"), nil, log)
	}
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceName = steps.CreateKymaNameFromOperation(operation)
		op.RuntimeResourceName = steps.KymaRuntimeResourceName(operation)
	}, log)
}
