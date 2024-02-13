package deprovisioning

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
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
	return &DeleteGardenerClusterStep{
		operationManager: process.NewOperationManager(operations),
		kcpClient:        kcpClient,
		instances:        instances,
	}
}

func (step *DeleteGardenerClusterStep) Name() string {
	return "Delete_GardenerCluster"
}

func (step *DeleteGardenerClusterStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	namespace := operation.KymaResourceNamespace
	if namespace == "" {
		logger.Warnf("namespace for Kyma resource not specified, setting 'kcp-system'")
		namespace = "kcp-system"
	}
	resourceName := operation.GardenerClusterName
	if resourceName == "" {
		logger.Infof("GardenerCluster resource name is empty, using runtime-id")
		resourceName = steps.GardenerClusterName(&operation)
	}
	if resourceName == "" {
		logger.Infof("Using instance.RuntimeID")
		instance, err := step.instances.GetByID(operation.InstanceID)
		if err != nil {
			logger.Errorf("Unable to get instance: %s", err.Error())
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to get instance", 15*time.Second, 2*time.Minute, logger)
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

	logger.Infof("Deleting GardenerCluster resource: %s in namespace:%s", operation.GardenerClusterName, operation.KymaResourceNamespace)

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
			logger.Errorf("unable to delete the GardenerCluster resource: %s", err)
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to delete the GardenerCluster resource", backoffForK8SOperation, timeoutForK8sOperation, logger)
		}
	}

	return operation, 0, nil
}
