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
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), nil, log)
	}

	if operation.RuntimeID == "" {
		return s.operationManager.OperationFailed(operation, fmt.Sprint("RuntimeID not set, cannot create Kyma name"), nil, log)
	}

	// could be simplified but this provides single source of truth for Kyma name
	operation.KymaResourceName = ""
	operation.KymaResourceName = steps.KymaName(operation)

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.RuntimeID = operation.RuntimeID
	}, log)
}
