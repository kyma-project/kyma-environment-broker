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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
	configProvider   config.ConfigMapConfigProvider
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations, configProvider config.ConfigMapConfigProvider) *InitKymaTemplate {
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
	planName, found := broker.AvailablePlans.GetPlanNameByID(broker.PlanIDType(operation.ProvisioningParameters.PlanID))
	if !found {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("PlanID %s not found in PlanNamesMapping", operation.ProvisioningParameters.PlanID), nil, logger)
	}
	cfg := &internal.ConfigForPlan{}
	err := s.configProvider.Provide(planName, cfg)
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to provide configuration for plan %s", planName), err, 10*time.Second, 30*time.Second, logger)
	}
	obj, err := DecodeKymaTemplate(cfg.KymaTemplate)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to create kyma template: %s", err.Error()))
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}

	kymaTemplate, err := s.applyChannelOverride(cfg.KymaTemplate, obj, operation, planName, logger)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "unable to apply channel override", err, logger)
	}

	logger.Info(fmt.Sprintf("Decoded kyma template: %v", obj))
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = kymaTemplate
	}, logger)
}

func (s *InitKymaTemplate) applyChannelOverride(defaultTemplate string, obj *unstructured.Unstructured, operation internal.Operation, planName string, logger *slog.Logger) (string, error) {
	if operation.ProvisioningParameters.Parameters.Modules == nil || operation.ProvisioningParameters.Parameters.Modules.Channel == nil {
		logger.Info(fmt.Sprintf("Using default channel from ConfigMap for plan %s", planName))
		return defaultTemplate, nil
	}

	userChannel := *operation.ProvisioningParameters.Parameters.Modules.Channel

	if userChannel != "fast" && userChannel != "regular" {
		return "", fmt.Errorf("invalid channel value: %s. Allowed values are 'fast' or 'regular'", userChannel)
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return "", fmt.Errorf("unable to get spec from kyma template: %w", err)
	}
	if !found {
		return "", fmt.Errorf("spec not found in kyma template")
	}

	spec["channel"] = userChannel
	if err := unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
		return "", fmt.Errorf("unable to set channel in kyma template: %w", err)
	}

	kymaTemplate, err := EncodeKymaTemplate(obj)
	if err != nil {
		return "", fmt.Errorf("unable to encode kyma template with overridden channel: %w", err)
	}

	logger.Info(fmt.Sprintf("Applied user-specified channel '%s' to kyma template for plan %s", userChannel, planName))
	return kymaTemplate, nil
}
