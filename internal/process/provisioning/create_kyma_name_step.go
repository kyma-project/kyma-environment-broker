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

type CreateKymaNameStep struct {
	operationManager *process.OperationManager
}

func NewCreateKymaNameStep(os storage.Operations) *CreateKymaNameStep {
	return &CreateKymaNameStep{
		operationManager: process.NewOperationManager(os),
	}
}

func (s *CreateKymaNameStep) Name() string {
	return "Create_Kyma_Name"
}

func (s *CreateKymaNameStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		return s.operationManager.OperationFailed(operation, fmt.Sprint("RuntimeID not set, cannot create Kyma name"), nil, log)
	}

	operation.KymaResourceName = steps.CreateKymaNameFromOperation(operation)

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.RuntimeID = operation.RuntimeID
	}, log)
}
