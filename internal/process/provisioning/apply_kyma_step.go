package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	kebErr "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplyKymaStep struct {
	operationManager *process.OperationManager
	k8sClient        client.Client
}

var _ process.Step = &ApplyKymaStep{}

func NewApplyKymaStep(os storage.Operations, cli client.Client) *ApplyKymaStep {
	step := &ApplyKymaStep{k8sClient: cli}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebErr.LifeCycleManagerDependency)
	return step
}

func (a *ApplyKymaStep) Name() string {
	return "Apply_Kyma"
}

func (a *ApplyKymaStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	template, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return a.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	a.addLabelsAndName(operation, template)
	operation, backoff, _ := a.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceName = template.GetName()
	}, logger)
	if backoff != 0 {
		logger.Error("cannot save the operation")
		return operation, 5 * time.Second, nil
	}

	var existingKyma unstructured.Unstructured
	existingKyma.SetGroupVersionKind(template.GroupVersionKind())
	err = a.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: operation.KymaResourceNamespace,
		Name:      template.GetName(),
	}, &existingKyma)

	switch {
	case err == nil:
		logger.Info(fmt.Sprintf("Kyma resource already exists, updating Kyma resource: %s in namespace %s", existingKyma.GetName(), existingKyma.GetNamespace()))
		changed := a.addLabelsAndName(operation, &existingKyma)
		if !changed {
			logger.Info("Kyma resource does not need any change")
		}
		err = a.k8sClient.Update(context.Background(), &existingKyma)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to update a Kyma resource: %s", err.Error()))
			return a.operationManager.RetryOperation(operation, "unable to update the Kyma resource", err, time.Second, 10*time.Second, logger)
		}
	case errors.IsNotFound(err):
		logger.Info(fmt.Sprintf("creating Kyma resource: %s in namespace: %s", template.GetName(), template.GetNamespace()))
		err := a.k8sClient.Create(context.Background(), template)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to create a Kyma resource: %s", err.Error()))
			return a.operationManager.RetryOperation(operation, "unable to create the Kyma resource", err, time.Second, 10*time.Second, logger)
		}
	default:
		logger.Error(fmt.Sprintf("Unable to get Kyma: %s", err.Error()))
		return a.operationManager.RetryOperation(operation, "unable to get the Kyma resource", err, time.Second, 10*time.Second, logger)
	}

	return operation, 0, nil
}

func (a *ApplyKymaStep) addLabelsAndName(operation internal.Operation, obj *unstructured.Unstructured) bool {
	oldLabels := obj.GetLabels()
	steps.ApplyLabelsAndAnnotationsForLM(obj, operation)

	// Kyma resource name must be created once again
	obj.SetName(steps.CreateKymaNameFromOperation(operation))
	return !reflect.DeepEqual(obj.GetLabels(), oldLabels)
}
