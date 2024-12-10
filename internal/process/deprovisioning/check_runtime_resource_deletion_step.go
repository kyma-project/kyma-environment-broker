package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CheckRuntimeResourceDeletionStep struct {
	operationManager                        *process.OperationManager
	kcpClient                               client.Client
	checkRuntimeResourceDeletionStepTimeout time.Duration
}

func NewCheckRuntimeResourceDeletionStep(operations storage.Operations, kcpClient client.Client, checkRuntimeResourceDeletionStepTimeout time.Duration) *CheckRuntimeResourceDeletionStep {
	step := &CheckRuntimeResourceDeletionStep{
		kcpClient:                               kcpClient,
		checkRuntimeResourceDeletionStepTimeout: checkRuntimeResourceDeletionStepTimeout,
	}
	step.operationManager = process.NewOperationManager(operations, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (step *CheckRuntimeResourceDeletionStep) Name() string {
	return "Check_RuntimeResource_Deletion"
}

func (step *CheckRuntimeResourceDeletionStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	namespace := operation.KymaResourceNamespace
	if namespace == "" {
		logger.Warn("namespace for Kyma resource not specified, setting 'kcp-system'")
		namespace = "kcp-system"
	}
	resourceName := operation.RuntimeResourceName
	if resourceName == "" {
		logger.Info("Runtime resource name is empty, using runtime-id")
		resourceName = operation.RuntimeID
	}
	if resourceName == "" {
		logger.Info("Empty runtime ID, skipping")
		return operation, 0, nil
	}

	runtime := &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
	}

	err := step.kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      resourceName,
	}, runtime)

	if err == nil {
		logger.Info("Runtime resource still exists")
		return step.operationManager.RetryOperation(operation, "Runtime resource still exists", nil, 20*time.Second, step.checkRuntimeResourceDeletionStepTimeout, logger)
	}

	if !errors.IsNotFound(err) {
		if meta.IsNoMatchError(err) {
			logger.Info("No CRD installed, skipping")
			return operation, 0, nil
		}

		logger.Warn(fmt.Sprintf("unable to check Runtime resource existence: %s", err))
		return step.operationManager.RetryOperation(operation, "unable to check Runtime resource existence", err, backoffForK8SOperation, timeoutForK8sOperation, logger)
	}

	return step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.RuntimeResourceName = ""
	}, logger)
}
