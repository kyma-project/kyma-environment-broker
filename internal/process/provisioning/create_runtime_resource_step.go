package provisioning

import (
	"fmt"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/kim"
	"gopkg.in/yaml.v3"

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

	runtimeCR, err := s.createRuntimeResourceObject(operation)
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
	}

	if s.kimConfig.DryRun {
		yaml, err := RuntimeToYaml(runtimeCR)
		if err != nil {
			log.Errorf("failed to encode Runtime CR to yaml: %s", err)
		} else {
			log.Infof("Runtime CR yaml:%s", yaml)
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

func (s *CreateRuntimeResourceStep) createRuntimeResourceObject(operation internal.Operation) (*imv1.Runtime, error) {
	runtime := imv1.Runtime{}
	runtime.Spec.Shoot.Name = "shoot-name"

	return &runtime, nil
}

func RuntimeToYaml(runtime *imv1.Runtime) (string, error) {
	result, err := yaml.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
