package steps

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations) *InitKymaTemplate {
	step := &InitKymaTemplate{}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	tmpl := operation.InputCreator.Configuration().KymaTemplate
	obj, err := DecodeKymaTemplate(tmpl)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to create kyma template: %s", err.Error()))
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	logger.Info(fmt.Sprintf("Decoded kyma template: %v", obj))
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = tmpl
	}, logger)
}
