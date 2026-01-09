package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/config"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

const (
	backoffForK8SOperation = time.Second
	timeoutForK8sOperation = 10 * time.Second
)

type DeleteKymaResourceStep struct {
	operationManager *process.OperationManager
	instances        storage.Instances
	kcpClient        client.Client
	configProvider   config.ConfigMapConfigProvider
}

func NewDeleteKymaResourceStep(db storage.BrokerStorage, kcpClient client.Client, configProvider config.ConfigMapConfigProvider) *DeleteKymaResourceStep {
	step := &DeleteKymaResourceStep{
		instances:      db.Instances(),
		kcpClient:      kcpClient,
		configProvider: configProvider,
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.LifeCycleManagerDependency)
	return step
}

func (step *DeleteKymaResourceStep) Name() string {
	return "Delete_Kyma_Resource"
}

func (step *DeleteKymaResourceStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	// read the KymaTemplate from the config if needed
	if operation.KymaTemplate == "" {
		cfg := &internal.ConfigForPlan{}
		err := step.configProvider.Provide(broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(operation.ProvisioningParameters.PlanID)), cfg) // TODO check if this is properly passed
		if err != nil {
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to get config for given version and plan", 5*time.Second, 30*time.Second, logger,
				fmt.Errorf("unable to get config for given version and plan"))
		}
		modifiedOperation, backoff, err := step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.KymaTemplate = cfg.KymaTemplate
		}, logger)
		if backoff > 0 {
			return operation, backoff, err
		}
		operation = modifiedOperation
	}
	obj, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to decode kyma template", 5*time.Second, 30*time.Second, logger,
			fmt.Errorf("unable to decode kyma template"))
	}

	if operation.KymaResourceNamespace == "" {
		logger.Warn("namespace for Kyma resource not specified")
		return operation, 0, nil
	}
	kymaResourceName := steps.KymaName(operation)
	if kymaResourceName == "" {
		logger.Info("Kyma resource name is empty, using instance.RuntimeID")

		instance, err := step.instances.GetByID(operation.InstanceID)
		if err != nil {
			logger.Warn(fmt.Sprintf("Unable to get instance: %s", err.Error()))
			return step.operationManager.RetryOperationWithoutFail(operation, err.Error(), "unable to get instance", 15*time.Second, 2*time.Minute, logger, err)
		}
		kymaResourceName = steps.KymaNameFromInstance(instance)
		// save the kyma resource name if it was taken from the instance.runtimeID
		backoff := time.Duration(0)
		operation, backoff, _ = step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.KymaResourceName = kymaResourceName
		}, logger)
		if backoff > 0 {
			return operation, backoff, nil
		}
	}
	if kymaResourceName == "" {
		logger.Info("KymaResourceName is empty, skipping")
		return operation, 0, nil
	}

	logger.Info(fmt.Sprintf("Deleting Kyma resource: %s in namespace:%s", kymaResourceName, operation.KymaResourceNamespace))

	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetName(kymaResourceName)
	kymaUnstructured.SetNamespace(operation.KymaResourceNamespace)
	kymaUnstructured.SetGroupVersionKind(obj.GroupVersionKind())

	err = step.kcpClient.Delete(context.Background(), kymaUnstructured)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("no Kyma resource to delete - ignoring")
		} else {
			logger.Warn(fmt.Sprintf("unable to delete the Kyma resource: %s", err))
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to delete the Kyma resource", backoffForK8SOperation, timeoutForK8sOperation, logger, err)
		}
	}

	return operation, 0, nil
}
