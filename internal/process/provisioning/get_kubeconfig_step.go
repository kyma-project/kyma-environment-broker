package provisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type GetKubeconfigStep struct {
	provisionerClient   provisioner.Client
	operationManager    *process.OperationManager
	provisioningTimeout time.Duration
	kimConfig           broker.KimConfig
}

func NewGetKubeconfigStep(os storage.Operations,
	provisionerClient provisioner.Client,
	kimConfig broker.KimConfig) *GetKubeconfigStep {
	step := &GetKubeconfigStep{
		provisionerClient: provisionerClient,
		kimConfig:         kimConfig,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.ProvisionerDependency)
	return step
}

var _ process.Step = (*GetKubeconfigStep)(nil)

func (s *GetKubeconfigStep) Name() string {
	return "Get_Kubeconfig"
}

func (s *GetKubeconfigStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *GetKubeconfigStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {

	if s.kimConfig.IsDrivenByKimOnly(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		log.Info(fmt.Sprintf("KIM is driving the process for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]))
		return operation, 0, nil
	}

	if operation.Kubeconfig == "" {
		if broker.IsOwnClusterPlan(operation.ProvisioningParameters.PlanID) {
			operation.Kubeconfig = operation.ProvisioningParameters.Parameters.Kubeconfig
		} else {
			if operation.RuntimeID == "" {
				log.Error("Runtime ID is empty")
				return s.operationManager.OperationFailed(operation, "Runtime ID is empty", nil, log)
			}
			kubeconfigFromRuntimeStatus, backoff, err := s.getKubeconfigFromRuntimeStatus(operation, log)
			if backoff > 0 {
				return operation, backoff, err
			}
			operation.Kubeconfig = kubeconfigFromRuntimeStatus
		}
	}

	return operation, 0, nil
}

func (s *GetKubeconfigStep) getKubeconfigFromRuntimeStatus(operation internal.Operation, log *slog.Logger) (string, time.Duration, error) {

	status, err := s.provisionerClient.RuntimeStatus(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID)
	if err != nil {
		log.Error(fmt.Sprintf("call to provisioner RuntimeStatus failed: %s", err.Error()))
		return "", 1 * time.Minute, nil
	}

	if status.RuntimeConfiguration.Kubeconfig == nil {
		log.Error("kubeconfig is not provided")
		return "", 1 * time.Minute, nil
	}

	kubeconfig := *status.RuntimeConfiguration.Kubeconfig

	log.Info(fmt.Sprintf("kubeconfig details length: %v", len(kubeconfig)))
	if len(kubeconfig) < 10 {
		log.Info(fmt.Sprintf("kubeconfig details length: %v", len(kubeconfig)))
		return "", 30 * time.Second, nil
	}

	return kubeconfig, 0, nil
}
