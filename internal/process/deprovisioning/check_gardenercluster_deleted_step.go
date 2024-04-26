package deprovisioning

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type CheckGardenerClusterDeletedStep struct {
	operationManager *process.OperationManager
	kcpClient        client.Client
}

func NewCheckGardenerClusterDeletedStep(operations storage.Operations, kcpClient client.Client) *CheckGardenerClusterDeletedStep {
	return &CheckGardenerClusterDeletedStep{
		operationManager: process.NewOperationManager(operations),
		kcpClient:        kcpClient,
	}
}

func (step *CheckGardenerClusterDeletedStep) Name() string {
	return "Check_GardenerCluster_Deleted"
}

func (step *CheckGardenerClusterDeletedStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
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
		logger.Infof("Empty runtime ID, skipping")
		return operation, 0, nil
	}

	gardenerClusterUnstructured := &unstructured.Unstructured{}
	gardenerClusterUnstructured.SetGroupVersionKind(steps.GardenerClusterGVK())
	err := step.kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      resourceName,
	}, gardenerClusterUnstructured)

	if err == nil {
		logger.Infof("GardenerCluster resource still exists")
		//TODO: extract the timeout as a configuration setting
		return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "GardenerCluster resource still exists", 5*time.Second, 1*time.Minute, logger, nil)
	}

	if !errors.IsNotFound(err) {
		if meta.IsNoMatchError(err) {
			logger.Info("No CRD installed, skipping")
			return operation, 0, nil
		}

		logger.Warnf("unable to check GardenerCluster resource existence: %s", err)
		return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to check GardenerCluster resource existence", backoffForK8SOperation, timeoutForK8sOperation, logger, err)
	}

	return step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.GardenerClusterName = ""
	}, logger)
}
