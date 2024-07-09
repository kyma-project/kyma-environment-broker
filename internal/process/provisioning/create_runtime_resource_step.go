package provisioning

import (
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/kim"
	"gopkg.in/yaml.v3"
	"time"

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
	}

	runtimeCR, err := s.createRuntimeResourceObject(operation)
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
	}

	if s.kimConfig.DryRun {
		yaml, err := EncodeRuntimeCR(runtimeCR)
		if err != nil {
			log.Infof("Runtime CR yaml:%s", yaml)
		} else {
			log.Errorf("failed to encode Runtime CR to yaml: %s", err)
		}
	} else {
		err := s.CreateResource(runtimeCR)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)

		}
	}
	log.Info("Runtime CR creation process finished successfully")
	return operation, 0, nil
}

func (s *CreateRuntimeResourceStep) CreateResource(cr imv1.Runtime) error {
	return nil
}

//TODO remember - labels and annotations

func (s *CreateRuntimeResourceStep) createRuntimeResourceObject(operation internal.Operation) (imv1.Runtime, error) {
	runtime := imv1.Runtime{}

	//operation.InputCreator.SetProvisioningParameters(operation.ProvisioningParameters)
	//operation.InputCreator.SetShootName(operation.ShootName)
	//operation.InputCreator.SetShootDomain(operation.ShootDomain)
	//operation.InputCreator.SetShootDNSProviders(operation.ShootDNSProviders)
	//operation.InputCreator.SetLabel(brokerKeyPrefix+"instance_id", operation.InstanceID)
	//operation.InputCreator.SetLabel(globalKeyPrefix+"subaccount_id", operation.ProvisioningParameters.ErsContext.SubAccountID)
	//operation.InputCreator.SetLabel(grafanaURLLabel, fmt.Sprintf("https://grafana.%s", operation.ShootDomain))
	//request, err, := operation.InputCreator.CreateProvisionClusterInput()
	//if err != nil {
	//	return imv1.Runtime{}, fmt.Errorf("while building input for provisioner: %w", err)
	//}
	//request.ClusterConfig.GardenerConfig.ShootNetworkingFilterDisabled = operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()
	//request.ClusterConfig.GardenerConfig.EuAccess = &operation.InstanceDetails.EuAccess

	return runtime, nil
}

func EncodeRuntimeCR(runtime imv1.Runtime) (string, error) {
	result, err := yaml.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
