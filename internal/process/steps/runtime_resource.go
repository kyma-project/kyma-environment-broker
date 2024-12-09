package steps

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCheckRuntimeResourceStep(os storage.Operations, k8sClient client.Client, kimConfig broker.KimConfig, runtimeResourceStepTimeout time.Duration) *checkRuntimeResource {
	step := &checkRuntimeResource{
		k8sClient:                  k8sClient,
		kimConfig:                  kimConfig,
		runtimeResourceStepTimeout: runtimeResourceStepTimeout,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

type checkRuntimeResource struct {
	k8sClient                  client.Client
	kimConfig                  broker.KimConfig
	operationManager           *process.OperationManager
	runtimeResourceStepTimeout time.Duration
}

func (_ *checkRuntimeResource) Name() string {
	return "Check_RuntimeResource"
}

func (s *checkRuntimeResource) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if !s.kimConfig.IsDrivenByKim(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		log.Info("Only provisioner is controlling provisioning process, skipping")
		return operation, 0, nil
	}

	runtime, err := s.GetRuntimeResource(operation.RuntimeID, operation.KymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get Runtime resource %s/%s", operation.KymaResourceNamespace, operation.RuntimeID))
		return s.operationManager.RetryOperation(operation, "unable to get Runtime resource", err, time.Second, 10*time.Second, log)
	}

	// check status
	state := runtime.Status.State
	log.Info(fmt.Sprintf("Runtime resource state: %s", state))
	if state != imv1.RuntimeStateReady {
		if time.Since(operation.CreatedAt) > s.runtimeResourceStepTimeout {
			description := fmt.Sprintf("Waiting for Runtime resource (%s/%s) ready state timeout.", operation.KymaResourceNamespace, operation.RuntimeID)
			log.Error(description)
			log.Info(fmt.Sprintf("Runtime resource status: %v, timeout: %v", runtime.Status, s.runtimeResourceStepTimeout))
			return s.operationManager.OperationFailed(operation, description, nil, log)
		} else {
			log.Info(fmt.Sprintf("Runtime resource status: %v, time since last update: %v, timeout set: %v", runtime.Status, time.Since(operation.UpdatedAt), s.runtimeResourceStepTimeout))
		}
		return operation, 10 * time.Second, nil
	}
	return operation, 0, nil
}

func (s *checkRuntimeResource) GetRuntimeResource(name string, namespace string) (*imv1.Runtime, error) {
	runtime := imv1.Runtime{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &runtime)
	if err != nil {
		return nil, err
	}
	return &runtime, nil
}
