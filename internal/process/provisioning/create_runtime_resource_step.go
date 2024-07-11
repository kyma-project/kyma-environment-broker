package provisioning

import (
	"fmt"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"time"

	"sigs.k8s.io/yaml"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/kim"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type CreateRuntimeResourceStep struct {
	operationManager    *process.OperationManager
	instanceStorage     storage.Instances
	runtimeStateStorage storage.RuntimeStates
	kimConfig           kim.Config
}

func NewCreateRuntimeResourceStep(os storage.Operations, runtimeStorage storage.RuntimeStates, is storage.Instances, kimConfig kim.Config) *CreateRuntimeResourceStep {
	return &CreateRuntimeResourceStep{
		operationManager:    process.NewOperationManager(os),
		instanceStorage:     is,
		runtimeStateStorage: runtimeStorage,
		kimConfig:           kimConfig,
	}
}

func (s *CreateRuntimeResourceStep) Name() string {
	return "Create_Runtime_Resource"
}

func (s *CreateRuntimeResourceStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), nil, log)
	}

	if !s.kimConfig.IsEnabledForPlan(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		log.Infof("KIM is not enabled for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID])
		return operation, 0, nil
	}

	template, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, log)
	}

	runtimeCR, err := s.createRuntimeResourceObject(operation, template.GetName(), template.GetNamespace())
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
	}

	if s.kimConfig.DryRun {
		yaml, err := RuntimeToYaml(runtimeCR)
		if err != nil {
			log.Errorf("failed to encode Runtime CR as yaml: %s", err)
		} else {
			fmt.Println(yaml)
		}
	} else {
		err := s.CreateResource(runtimeCR)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR resource: %s", err), err, log)
		}
		log.Info("Runtime CR creation process finished successfully")
	}
	return operation, 0, nil
}

func (s *CreateRuntimeResourceStep) CreateResource(cr *imv1.Runtime) error {
	logrus.Info("Creating Runtime CR - TO BE IMPLEMENTED")
	return nil
}

func (s *CreateRuntimeResourceStep) createRuntimeResourceObject(operation internal.Operation, kymaName, kymaNamespace string) (*imv1.Runtime, error) {

	runtime := imv1.Runtime{}
	runtime.ObjectMeta.Name = operation.RuntimeID
	runtime.ObjectMeta.Namespace = kymaNamespace
	runtime.ObjectMeta.Labels = s.createLabelsForRuntime(operation, kymaName)
	runtime.Spec.Shoot.Provider = s.createShootProvider(operation)
	runtime.Spec.Shoot.Provider.Workers = []gardener.Worker{}
	runtime.Spec.Shoot.Provider.Type = string(operation.ProvisioningParameters.PlatformProvider)
	runtime.Spec.Security = s.createSecurityConfiguration(operation)
	return &runtime, nil
}

func (s *CreateRuntimeResourceStep) createShootProvider(operation internal.Operation) imv1.Provider {
	provider := imv1.Provider{}
	logrus.Info("Creating Shoot Provider - TO BE IMPLEMENTED")
	return provider
}

func (s *CreateRuntimeResourceStep) createLabelsForRuntime(operation internal.Operation, kymaName string) map[string]string {
	labels := map[string]string{
		"kyma-project.io/instance-id":        operation.InstanceID,
		"kyma-project.io/runtime-id":         operation.RuntimeID,
		"kyma-project.io/broker-plan-id":     operation.ProvisioningParameters.PlanID,
		"kyma-project.io/broker-plan-name":   broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
		"kyma-project.io/global-account-id":  operation.ProvisioningParameters.ErsContext.GlobalAccountID,
		"kyma-project.io/subaccount-id":      operation.ProvisioningParameters.ErsContext.SubAccountID,
		"kyma-project.io/shoot-name":         operation.ShootName,
		"kyma-project.io/region":             *operation.ProvisioningParameters.Parameters.Region,
		"operator.kyma-project.io/kyma-name": kymaName,
	}
	return labels
}

func (s *CreateRuntimeResourceStep) createSecurityConfiguration(operation internal.Operation) imv1.Security {
	security := imv1.Security{}
	logrus.Info("Creating Security Configuration - TO BE IMPLEMENTED")
	return security
}

func RuntimeToYaml(runtime *imv1.Runtime) (string, error) {
	result, err := yaml.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
