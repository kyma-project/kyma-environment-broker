package update

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type InitKymaVersionStep struct {
	operationManager       *process.OperationManager
	runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator
	runtimeStatesDb        storage.RuntimeStates
}

func NewInitKymaVersionStep(os storage.Operations, rvc *runtimeversion.RuntimeVersionConfigurator, runtimeStatesDb storage.RuntimeStates) *InitKymaVersionStep {
	return &InitKymaVersionStep{
		operationManager:       process.NewOperationManager(os),
		runtimeVerConfigurator: rvc,
		runtimeStatesDb:        runtimeStatesDb,
	}
}

func (s *InitKymaVersionStep) Name() string {
	return "Update_Init_Kyma_Version"
}

func (s *InitKymaVersionStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	var version *internal.RuntimeVersionData
	var err error
	if operation.RuntimeVersion.IsEmpty() {
		version, err = s.runtimeVerConfigurator.ForUpdating(operation)
		if err != nil {
			return s.operationManager.RetryOperation(operation, "error while getting runtime version", err, 5*time.Second, 1*time.Minute, log)
		}
	} else {
		version = &operation.RuntimeVersion
	}
	op, delay, _ := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		if version != nil {
			op.RuntimeVersion = *version
		}
	}, log)
	log.Info("Init runtime version: ", op.RuntimeVersion.MajorVersion)
	return op, delay, nil
}