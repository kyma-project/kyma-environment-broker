package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteGardenerClusterStep struct {
	operationManager *process.OperationManager
	kcpClient        client.Client
	instances        storage.Instances
}

func NewDeleteGardenerClusterStep(operations storage.Operations, kcpClient client.Client, instances storage.Instances) *DeleteGardenerClusterStep {
	step := &DeleteGardenerClusterStep{
		kcpClient: kcpClient,
		instances: instances,
	}
	step.operationManager = process.NewOperationManager(operations, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (step *DeleteGardenerClusterStep) Name() string {
	return "Delete_GardenerCluster"
}

func (s *DeleteGardenerClusterStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (step *DeleteGardenerClusterStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	namespace := operation.KymaResourceNamespace
	if namespace == "" {
		logger.Warn("namespace for Kyma resource not specified, setting 'kcp-system'")
		namespace = "kcp-system"
	}
	resourceName := operation.GardenerClusterName
	if resourceName == "" {
		logger.Info("GardenerCluster resource name is empty, using runtime-id")
		resourceName = steps.GardenerClusterName(&operation)
	}
	if resourceName == "" {
		logger.Info("Using instance.RuntimeID")
		instance, err := step.instances.GetByID(operation.InstanceID)
		if err != nil {
			logger.Warn(fmt.Sprintf("Unable to get instance: %s", err.Error()))
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to get instance", 15*time.Second, 2*time.Minute, logger, err)
		}
		resourceName = steps.GardenerClusterNameFromInstance(instance)

		// save the gardener cluster resource name to use for checking step
		backoff := time.Duration(0)
		operation, backoff, _ = step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.GardenerClusterName = resourceName
		}, logger)
		if backoff > 0 {
			return operation, backoff, nil
		}
	}

	if resourceName == "" {
		logger.Info("Gardener cluster not known, skipping")
		return operation, 0, nil
	}

	logger.Info(fmt.Sprintf("Deleting GardenerCluster resource: %s in namespace:%s", operation.GardenerClusterName, operation.KymaResourceNamespace))

	gardenerClusterUnstructured := &unstructured.Unstructured{}
	gardenerClusterUnstructured.SetName(resourceName)
	gardenerClusterUnstructured.SetNamespace(namespace)
	gardenerClusterUnstructured.SetGroupVersionKind(steps.GardenerClusterGVK())

	err := step.kcpClient.Delete(context.Background(), gardenerClusterUnstructured)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("no GardenerCluster resource to delete - ignoring")
		} else if meta.IsNoMatchError(err) {
			logger.Info("No CRD installed, skipping")
		} else {
			logger.Warn(fmt.Sprintf("unable to delete the GardenerCluster resource: %s", err))
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to delete the GardenerCluster resource", backoffForK8SOperation, timeoutForK8sOperation, logger, err)
		}
	}

	return operation, 0, nil
}
