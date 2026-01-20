package provisioning

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type OverrideKymaModules struct {
	operationManager *process.OperationManager
	logger           *slog.Logger
}

var _ process.Step = &OverrideKymaModules{}

func (k *OverrideKymaModules) Name() string {
	return "Override_Kyma_Modules"
}

func NewOverrideKymaModules(os storage.Operations) *OverrideKymaModules {
	step := &OverrideKymaModules{}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

// Cases:
// 1 case -> if 'default' is false, then we don't install anything, no modules
// 2 case -> if 'list' is given and not empty, we override passed modules
// 3 case -> if 'list' is given and is empty, then we don't install anything, no modules
// default behaviour is when default = true, then default modules will be installed, also it applies to all other scenarios than mentioned in 1,2,3 point.

func (k *OverrideKymaModules) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	k.logger = logger
	if operation.Type != internal.OperationTypeProvision {
		k.logger.Info(fmt.Sprintf("%s is supposed to run only for Provisioning, skipping logic.", k.Name()))
		return operation, 0, nil
	}

	modulesParams := operation.ProvisioningParameters.Parameters.Modules
	if modulesParams != nil {
		defaultModulesSetToFalse := modulesParams.Default != nil && !*modulesParams.Default  // 1 case
		customModulesListPassed := modulesParams.Default == nil && modulesParams.List != nil // 2 & 3 case
		overrideModules := defaultModulesSetToFalse || customModulesListPassed
		if overrideModules {
			k.logger.Info("custom modules parameters are set, the content of list will replace current modules section. Default settings will be overridden.")
			return k.handleModulesOverride(operation, *modulesParams)
		}
	}

	// default behaviour
	k.logger.Info(fmt.Sprintf("Kyma will be created with default modules. %s didn't perform any action.", k.Name()))
	return operation, 0, nil
}

func (k *OverrideKymaModules) handleModulesOverride(operation internal.Operation, modulesParams pkg.ModulesDTO) (internal.Operation, time.Duration, error) {
	decodeKymaTemplate, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		k.logger.Error(fmt.Sprintf("while decoding Kyma template from previous step: %s", err.Error()))
		return k.operationManager.OperationFailed(operation, "while decoding Kyma template from previous step", err, k.logger)
	}
	if decodeKymaTemplate == nil {
		k.logger.Error("while decoding Kyma template from previous step: object is nil")
		return k.operationManager.OperationFailed(operation, "while decoding Kyma template from previous step: ", fmt.Errorf("object is nil"), k.logger)
	}

	if err := k.replaceModulesSpec(decodeKymaTemplate, modulesParams.List); err != nil {
		k.logger.Error(fmt.Sprintf("unable to append modules to Kyma template: %s", err.Error()))
		return k.operationManager.OperationFailed(operation, "unable to append modules to Kyma template:", err, k.logger)
	}
	updatedKymaTemplate, err := steps.EncodeKymaTemplate(decodeKymaTemplate)
	if err != nil {
		k.logger.Error(fmt.Sprintf("unable to create yaml Kyma template with custom modules: %s", err.Error()))
		return k.operationManager.OperationFailed(operation, "unable to create yaml Kyma template within added modules", err, k.logger)
	}

	k.logger.Info("encoded Kyma template with custom modules with success")
	return k.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = decodeKymaTemplate.GetNamespace()
		op.KymaTemplate = updatedKymaTemplate
	}, k.logger)
}

func (k *OverrideKymaModules) replaceModulesSpec(kymaTemplate *unstructured.Unstructured, customModuleList []pkg.ModuleDTO) error {
	toInsert := k.prepareModulesSection(customModuleList)
	toInsertMarshaled, err := json.Marshal(toInsert)
	if err != nil {
		return err
	}
	var toInsertUnmarshaled interface{}
	err = json.Unmarshal(toInsertMarshaled, &toInsertUnmarshaled)
	if err != nil {
		return err
	}
	err = unstructured.SetNestedField(kymaTemplate.Object, toInsertUnmarshaled, "spec", "modules")
	if err != nil {
		return err
	}
	k.logger.Info("custom modules replaced in Kyma template successfully.")
	return nil
}
func (k *OverrideKymaModules) prepareModulesSection(customModuleList []pkg.ModuleDTO) []pkg.ModuleDTO {
	// if field is "" convert it to nil to field will be not present in yaml
	mapIfNeeded := func(field *string) *string {
		if field != nil && *field == "" {
			return nil
		}
		return field
	}

	overridedModules := make([]pkg.ModuleDTO, 0)
	if customModuleList == nil || len(customModuleList) == 0 {
		k.logger.Info("empty (0 items) list with custom modules passed to KEB, 0 modules will be installed - default config will be ignored")
		return overridedModules
	}

	for _, customModule := range customModuleList {
		module := pkg.ModuleDTO{Name: customModule.Name}
		module.CustomResourcePolicy = mapIfNeeded(customModule.CustomResourcePolicy)
		module.Channel = mapIfNeeded(customModule.Channel)
		overridedModules = append(overridedModules, module)
	}

	k.logger.Info(fmt.Sprintf("not empty list with custom modules passed to KEB. Number of modules: %d", len(overridedModules)))
	return overridedModules
}
