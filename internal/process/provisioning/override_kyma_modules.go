package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type OverrideKymaModules struct {
	operationManager *process.OperationManager
	logger           logrus.FieldLogger
}

var _ process.Step = &OverrideKymaModules{}

func (k *OverrideKymaModules) Name() string {
	return "Override_Kyma_Modules"
}

func NewOverrideKymaModules(os storage.Operations) *OverrideKymaModules {
	return &OverrideKymaModules{operationManager: process.NewOperationManager(os)}
}

// Cases:
// 1 case -> if 'default' is false, then we don't install anything, no modules
// 2 case -> if 'list' is given and not empty, we override passed modules
// 3 case -> if 'list' is given and is empty, then we don't install anything, no modules
// default behaviour is when default = true, then default modules will be installed, also it applies to all other scenarios than mentioned in 1,2,3 point.

func (k *OverrideKymaModules) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	k.logger = logger
	if operation.Type != internal.OperationTypeProvision {
		k.logger.Infof("%s is supposed to run only for Provisioning, skipping logic.", k.Name())
		return operation, 0, nil
	}

	modulesParams := operation.ProvisioningParameters.Parameters.Modules
	if modulesParams != nil {
		defaultModulesSetToFalse := modulesParams.Default != nil && !*modulesParams.Default  // 1 case
		customModulesListPassed := modulesParams.Default == nil && modulesParams.List != nil // 2 & 3 case
		overrideModules := defaultModulesSetToFalse || customModulesListPassed
		if overrideModules {
			k.logger.Info("custom modules parameters are set, the content of list will replace current modules section. Default settings will be overriden.")
			return k.handleModulesOverride(operation, *modulesParams)
		}
	}

	// default behaviour
	k.logger.Infof("Kyma will be created with default modules. %s didn't perform any action. %s", k.Name())
	return operation, 0, nil
}

func (k *OverrideKymaModules) handleModulesOverride(operation internal.Operation, modulesParams internal.ModulesDTO) (internal.Operation, time.Duration, error) {
	decodeKymaTemplate, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		k.logger.Errorf("while decoding Kyma template from previous step: %s", err.Error())
		return k.operationManager.OperationFailed(operation, "while decoding Kyma template from previous step", err, k.logger)
	}
	if decodeKymaTemplate == nil {
		k.logger.Errorf("while decoding Kyma template from previous step: object is nil")
		return k.operationManager.OperationFailed(operation, "while decoding Kyma template from previous step: ", fmt.Errorf("object is nil"), k.logger)
	}

	if err := k.replaceModulesSpec(decodeKymaTemplate, modulesParams); err != nil {
		k.logger.Errorf("unable to append modules to Kyma template: %s", err.Error())
		return k.operationManager.OperationFailed(operation, "unable to append modules to Kyma template:", err, k.logger)
	}
	updatedKymaTemplate, err := steps.EncodeKymaTemplate(decodeKymaTemplate)
	if err != nil {
		k.logger.Errorf("unable to create yaml Kyma template with custom custom modules: %s", err.Error())
		return k.operationManager.OperationFailed(operation, "unable to create yaml Kyma template within added modules", err, k.logger)
	}

	k.logger.Info("encoded Kyma template with custom modules with success")
	return k.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = decodeKymaTemplate.GetNamespace()
		op.KymaTemplate = updatedKymaTemplate
	}, k.logger)
}

// To consider using -> unstructured.SetNestedSlice()
func (k *OverrideKymaModules) replaceModulesSpec(kymaTemplate *unstructured.Unstructured, modulesParams internal.ModulesDTO) error {
	const (
		specKey    = "spec"
		modulesKey = "modules"
	)

	content := kymaTemplate.Object
	specSection, ok := content[specKey]
	if !ok {
		return fmt.Errorf("getting spec content of kyma template")
	}
	spec, ok := specSection.(map[string]interface{})
	if !ok {
		return fmt.Errorf("converting spec of kyma template")
	}
	modulesSection, ok := spec[modulesKey]
	if !ok {
		return fmt.Errorf("getting modules content of kyma template")
	}

	if modulesParams.List == nil || len(modulesParams.List) == 0 {
		if modulesParams.List == nil {
			modulesParams.List = make([]*internal.ModuleDTO, 0)
		}
		k.logger.Info("empty list with custom modules passed to KEB, 0 modules will be installed - default config will be ignored")
	} else {
		k.logger.Info("not empty list with custom modules passed to KEB. Number of modules: %d", len(modulesParams.List))
	}

	modulesSection = modulesParams.List
	spec[modulesKey] = modulesSection
	kymaTemplate.Object[specKey] = specSection

	k.logger.Info("custom modules replaced in Kyma template successfully.")
	return nil
}
