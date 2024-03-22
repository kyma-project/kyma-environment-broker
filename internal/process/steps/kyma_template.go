package steps

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations) *InitKymaTemplate {
	return &InitKymaTemplate{operationManager: process.NewOperationManager(os)}
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	tmpl := operation.InputCreator.Configuration().KymaTemplate
	obj, err := DecodeKymaTemplate(tmpl)
	if err != nil {
		logger.Errorf("Unable to create kyma template: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	logger.Infof("Decoded kyma template: %v", obj)
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = tmpl
	}, logger)
}

// NOTE: adapter for upgrade_kyma which is currently not using shared staged_manager
type initKymaTemplateUpgradeKyma struct {
	*InitKymaTemplate
}

func InitKymaTemplateUpgradeKyma(os storage.Operations) initKymaTemplateUpgradeKyma {
	return initKymaTemplateUpgradeKyma{NewInitKymaTemplate(os)}
}

func (s initKymaTemplateUpgradeKyma) Run(o internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation, w, err := s.InitKymaTemplate.Run(o.Operation, logger)
	return internal.UpgradeKymaOperation{Operation: operation}, w, err
}
