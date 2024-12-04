package deprovisioning

import (
	"context"
	"fmt"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type CheckKymaResourceDeletedStep struct {
	operationManager            *process.OperationManager
	kcpClient                   client.Client
	kymaResourceDeletionTimeout time.Duration
}

func NewCheckKymaResourceDeletedStep(operations storage.Operations, kcpClient client.Client, kymaResourceDeletionTimeout time.Duration) *CheckKymaResourceDeletedStep {
	step := &CheckKymaResourceDeletedStep{
		kcpClient:                   kcpClient,
		kymaResourceDeletionTimeout: kymaResourceDeletionTimeout,
	}
	step.operationManager = process.NewOperationManager(operations, step.Name(), kebError.LifeCycleManagerDependency)
	return step
}

func (step *CheckKymaResourceDeletedStep) Name() string {
	return "Check_Kyma_Resource_Deleted"
}

func (step *CheckKymaResourceDeletedStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.KymaResourceNamespace == "" {
		logger.Warnf("namespace for Kyma resource not specified")
		return operation, 0, nil
	}
	kymaResourceName := steps.KymaName(operation)
	if kymaResourceName == "" {
		logger.Infof("Kyma resource name is empty, skipping")
		return operation, 0, nil
	}

	obj, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to decode kyma template", 5*time.Second, 30*time.Second, logger,
			fmt.Errorf("unable to decode kyma template"))
	}

	logger.Infof("Checking existence of Kyma resource: %s in namespace:%s", kymaResourceName, operation.KymaResourceNamespace)

	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetGroupVersionKind(obj.GroupVersionKind())
	err = step.kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: operation.KymaResourceNamespace,
		Name:      kymaResourceName,
	}, kymaUnstructured)

	if err != nil && !errors.IsNotFound(err) {
		logger.Errorf("unable to check Kyma resource existence: %s", err)
		return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to check Kyma resource existence", backoffForK8SOperation, timeoutForK8sOperation, logger, err)
	}

	if err == nil {
		logger.Infof("Kyma resource still exists")
	}

	return step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceName = ""
	}, logger)
}
