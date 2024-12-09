package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const (
	timeoutForRuntimeDeletion = 10 * time.Minute
)

type DeleteRuntimeResourceStep struct {
	operationManager *process.OperationManager
	kcpClient        client.Client
}

func NewDeleteRuntimeResourceStep(operations storage.Operations, kcpClient client.Client) *DeleteRuntimeResourceStep {
	step := &DeleteRuntimeResourceStep{
		kcpClient: kcpClient,
	}
	step.operationManager = process.NewOperationManager(operations, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (step *DeleteRuntimeResourceStep) Name() string {
	return "Delete_Runtime_Resource"
}

func (step *DeleteRuntimeResourceStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	resourceName := operation.RuntimeResourceName
	resourceNamespace := operation.KymaResourceNamespace

	// if the resource name stored in the operation is empty, try to get it from the RuntimeID (when it was created by KIM migration process, not by the KEB)
	if resourceName == "" {
		resourceName = steps.KymaRuntimeResourceName(operation)
	}
	if resourceName == "" {
		logger.Info("Runtime resource name is empty, skipping")
		return operation, 0, nil
	}
	if resourceNamespace == "" {
		logger.Warn("Namespace for Runtime resource not specified")
		return operation, 0, nil
	}

	var runtime = imv1.Runtime{}
	err := step.kcpClient.Get(context.Background(), client.ObjectKey{Name: resourceName, Namespace: resourceNamespace}, &runtime)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Warn(fmt.Sprintf("Unable to read runtime: %s", err))
			return step.operationManager.RetryOperation(operation, err.Error(), err, 5*time.Second, 1*time.Minute, logger)
		} else {
			logger.Info("Runtime resource already deleted")
			return operation, 0, nil
		}
	}

	// save the information about the controller of SKR (Provisioner or KIM) only once, when the first deprovisioning is triggered
	if operation.KimDeprovisionsOnly == nil {
		controlledByKimOnly := !runtime.IsControlledByProvisioner()
		newOperation, backoff, _ := step.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
			operation.KimDeprovisionsOnly = ptr.Bool(controlledByKimOnly)
		}, logger)
		if backoff > 0 {
			return newOperation, backoff, nil
		}
		operation = newOperation
	}

	err = step.kcpClient.Delete(context.Background(), &runtime)

	// check the error
	if err != nil {
		if meta.IsNoMatchError(err) {
			logger.Info("No CRD installed, skipping")
			return operation, 0, nil
		}

		// if the resource is not found, log it and return (it is not a problem)
		if errors.IsNotFound(err) {
			logger.Info("Runtime resource already deleted")
			return operation, 0, nil
		} else {
			logger.Warn(fmt.Sprintf("unable to delete the Runtime resource %s/%s: %s", runtime.Name, runtime.Namespace, err))
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to delete the Runtime resource", backoffForK8SOperation, timeoutForK8sOperation, logger, err)
		}
	}

	return operation, 0, nil
}
