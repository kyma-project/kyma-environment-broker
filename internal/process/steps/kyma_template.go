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
	planName, found := broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
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
	logger.Info(fmt.Sprintf("Decoded kyma template: %v", obj))

	updatedTemplate, err := s.applyChannelToTemplate(operation, obj, logger)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "unable to apply channel to template", err, logger)
	}

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = updatedTemplate
	}, logger)
}

func (s *InitKymaTemplate) applyChannelToTemplate(operation internal.Operation, obj *unstructured.Unstructured, logger *slog.Logger) (string, error) {
	modulesParams := operation.ProvisioningParameters.Parameters.Modules
	if modulesParams == nil || modulesParams.Channel == nil || *modulesParams.Channel == "" {
		return EncodeKymaTemplate(obj)
	}

	userChannel := *modulesParams.Channel

	if userChannel != "regular" && userChannel != "fast" {
		return "", fmt.Errorf("invalid channel '%s': only 'regular' and 'fast' are allowed", userChannel)
	}

	currentChannel, found, err := unstructured.NestedString(obj.Object, "spec", "channel")
	if err != nil {
		return "", fmt.Errorf("failed to read current template channel: %w", err)
	}

	if found && currentChannel == userChannel {
		logger.Info(fmt.Sprintf("template channel '%s' matches user channel, no change needed", currentChannel))
	} else {
		err = unstructured.SetNestedField(obj.Object, userChannel, "spec", "channel")
		if err != nil {
			return "", fmt.Errorf("failed to set channel in template: %w", err)
		}
		if found {
			logger.Info(fmt.Sprintf("applied user channel '%s' to template (was '%s')", userChannel, currentChannel))
		} else {
			logger.Info(fmt.Sprintf("applied user channel '%s' to template (no previous channel)", userChannel))
		}
	}

	return EncodeKymaTemplate(obj)
}
