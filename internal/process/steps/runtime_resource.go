package steps

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/pivotal-cf/brokerapi/v12/domain"

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

func NewCheckRuntimeResourceProvisioningStep(os storage.Operations, k8sClient client.Client, runtimeResourceStateRetry internal.RetryTuple, changeDescriptionThreshold time.Duration) *checkRuntimeResourceProvisioning {
	step := &checkRuntimeResourceProvisioning{
		k8sClient:                  k8sClient,
		runtimeResourceStateRetry:  runtimeResourceStateRetry,
		changeDescriptionThreshold: changeDescriptionThreshold,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

type checkRuntimeResource struct {
	k8sClient                 client.Client
	operationManager          *process.OperationManager
	runtimeResourceStateRetry internal.RetryTuple
}

type checkRuntimeResourceProvisioning struct {
	k8sClient                  client.Client
	operationManager           *process.OperationManager
	runtimeResourceStateRetry  internal.RetryTuple
	changeDescriptionThreshold time.Duration
}

const (
	kcpRetryInterval = 3 * time.Second
	kcpRetryTimeout  = 20 * time.Second
)

func (_ *checkRuntimeResource) Name() string {
	return "Check_RuntimeResource_Update"
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

func (_ *checkRuntimeResourceProvisioning) Name() string {
	return "Check_RuntimeResource_Provisioning"
}

func (s *checkRuntimeResourceProvisioning) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	runtime, err := s.GetRuntimeResource(operation.RuntimeID, operation.KymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get Runtime resource %s/%s", operation.KymaResourceNamespace, operation.RuntimeID))
		return s.operationManager.RetryOperation(operation, "unable to get Runtime resource", err, kcpRetryInterval, kcpRetryTimeout, log)
	}

	// check status
	state := runtime.Status.State
	log.Info(fmt.Sprintf("Runtime resource state: %s", state))
	if state == imv1.RuntimeStateReady {
		return operation, 0, nil
	} else {
		if time.Since(operation.CreatedAt) > s.changeDescriptionThreshold {
			var backoff time.Duration
			operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
				op.Description = ProvisioningTakesLongerMessage(s.runtimeResourceStateRetry.Timeout)
			}, log)
			if backoff != 0 {
				log.Error("cannot save the operation")
				return operation, 5 * time.Second, nil
			}
		}
		return s.RetryOrFail(operation, log, runtime)
	}
}

func (s *checkRuntimeResourceProvisioning) RetryOrFail(operation internal.Operation, log *slog.Logger, runtime *imv1.Runtime) (internal.Operation, time.Duration, error) {
	retryOperation, retry, err := s.operationManager.RetryOperationWithCreatedAt(operation, fmt.Sprintf("Runtime resource not in %s state", imv1.RuntimeStateReady), nil, s.runtimeResourceStateRetry.Interval, s.runtimeResourceStateRetry.Timeout, log)
	if retryOperation.State == domain.Failed {
		log.Error(fmt.Sprintf("runtime resource state: %s", runtime.Status.State))
		log.Error(fmt.Sprintf("runtime resource provisioningCompleted: %v", runtime.Status.ProvisioningCompleted))
		for i, c := range runtime.Status.Conditions {
			log.Error(fmt.Sprintf(
				"runtime resource condition #%d: Type=%s, Status=%s, LastTransitionTime=%s, Reason=%s, Message=%s",
				i+1,
				c.Type,
				c.Status,
				c.LastTransitionTime.Format(time.RFC3339),
				c.Reason,
				c.Message,
			))
		}
		log.Error("failing operation and removing Runtime CR")
		err = s.k8sClient.Delete(context.Background(), runtime)
		if err != nil {
			log.Warn(fmt.Sprintf("unable to delete Runtime resource %s/%s: %s", runtime.Name, runtime.Namespace, err))
		}
	}
	return retryOperation, retry, err
}

func (s *checkRuntimeResource) GetRuntimeResource(name string, namespace string) (*imv1.Runtime, error) {
	return GetRuntimeResource(name, namespace, s.k8sClient)
}

func (s *checkRuntimeResourceProvisioning) GetRuntimeResource(name string, namespace string) (*imv1.Runtime, error) {
	return GetRuntimeResource(name, namespace, s.k8sClient)
}

func GetRuntimeResource(name string, namespace string, c client.Client) (*imv1.Runtime, error) {
	runtime := imv1.Runtime{}
	err := c.Get(context.Background(), client.ObjectKey{
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
	return config.IngressFilteringPlans.Contains(broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(planID))) && !external
}

func ProvisioningTakesLongerMessage(changeDescriptionThreshold time.Duration) string {
	return fmt.Sprintf("Operation created. Cluster provisioning takes longer than usual. It takes up to %d minutes.", int(changeDescriptionThreshold.Minutes()))
}
