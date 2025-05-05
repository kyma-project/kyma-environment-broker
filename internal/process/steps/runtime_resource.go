package steps

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCheckRuntimeResourceStep(os storage.Operations, k8sClient client.Client, runtimeResourceStateRetry internal.RetryTuple) *checkRuntimeResource {
	step := &checkRuntimeResource{
		k8sClient:                 k8sClient,
		runtimeResourceStateRetry: runtimeResourceStateRetry,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

type checkRuntimeResource struct {
	k8sClient                 client.Client
	operationManager          *process.OperationManager
	runtimeResourceStateRetry internal.RetryTuple
}

const (
	kcpRetryInterval = 3 * time.Second
	kcpRetryTimeout  = 20 * time.Second
)

func (_ *checkRuntimeResource) Name() string {
	return "Check_RuntimeResource"
}

func (s *checkRuntimeResource) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	runtime, err := s.GetRuntimeResource(operation.RuntimeID, operation.KymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get Runtime resource %s/%s", operation.KymaResourceNamespace, operation.RuntimeID))
		return s.operationManager.RetryOperation(operation, "unable to get Runtime resource", err, kcpRetryInterval, kcpRetryTimeout, log)
	}

	// check status
	state := runtime.Status.State
	log.Info(fmt.Sprintf("Runtime resource state: %s", state))
	switch state {
	case imv1.RuntimeStateReady:
		return operation, 0, nil
	case imv1.RuntimeStateFailed:
		log.Info(fmt.Sprintf("Runtime resource status: %v; failing operation", runtime.Status))
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Runtime resource in %s state", imv1.RuntimeStateFailed), nil, log)
	default:
		log.Info(fmt.Sprintf("Runtime resource status: %v; retrying in %v steps for: %v", runtime.Status, s.runtimeResourceStateRetry.Interval, s.runtimeResourceStateRetry.Timeout))
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("Runtime resource not in %s state", imv1.RuntimeStateReady), nil, s.runtimeResourceStateRetry.Interval, s.runtimeResourceStateRetry.Timeout, log)
	}
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

func IsNotSapConvergedCloud(cloudProvider string) bool {
	return cloudProvider != string(pkg.SapConvergedCloud)
}

func IsIngressFilteringEnabled(planID string, config broker.InfrastructureManager, external bool) bool {
	ingressFiltering := config.EnableIngressFiltering &&
		config.IngressFilteringPlans.Contains(broker.PlanNamesMapping[planID]) &&
		!external
	return ingressFiltering
}
