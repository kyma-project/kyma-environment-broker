package steps

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/config"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
	configProvider   config.ConfigurationProvider
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations, configProvider config.ConfigurationProvider) *InitKymaTemplate {
	step := &InitKymaTemplate{
		configProvider: configProvider,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	planName, found := broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	if !found {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("PlanID %s not found in PlanNamesMapping", operation.ProvisioningParameters.PlanID), nil, logger)
	}
	config, err := s.configProvider.ProvideForGivenPlan(planName)
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to provide configuration for plan %s", planName), err, 10*time.Second, 30*time.Second, logger)
	}
	obj, err := DecodeKymaTemplate(config.KymaTemplate)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to create kyma template: %s", err.Error()))
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	logger.Info(fmt.Sprintf("Decoded kyma template: %v", obj))
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = config.KymaTemplate
	}, logger)
}
