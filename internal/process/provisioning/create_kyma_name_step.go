package provisioning

import (
	"fmt"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"time"

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

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		operation.KymaResourceName = steps.CreateKymaNameFromOperation(operation)
	}, log)
}
