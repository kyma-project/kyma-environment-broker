package update

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UpdateKymaStep struct {
	operationManager *process.OperationManager
	kcpClient        client.Client
	instances        storage.Instances
	configProvider   config.ConfigMapConfigProvider
}

func NewUpdateKymaStep(db storage.BrokerStorage, kcpClient client.Client, configProvider config.ConfigMapConfigProvider) *UpdateKymaStep {
	step := &UpdateKymaStep{
		instances:      db.Instances(),
		kcpClient:      kcpClient,
		configProvider: configProvider,
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.LifeCycleManagerDependency)
	return step
}

func (s *UpdateKymaStep) Name() string {
	return "Update_Kyma_Resource"
}

func (s *UpdateKymaStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.UpdatedPlanID == "" {
		log.Info("Plan did not change, skipping update Kyma resource step")
		return operation, 0, nil
	}

	// read the KymaTemplate from the config if needed
	if operation.KymaTemplate == "" {
		cfg := &internal.ConfigForPlan{}
		err := s.configProvider.Provide(broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(operation.ProvisioningParameters.PlanID)), cfg)
		if err != nil {
			return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), "unable to get config for given version and plan", 5*time.Second, 30*time.Second, log,
				fmt.Errorf("unable to get config for given version and plan"))
		}
		modifiedOperation, backoff, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.KymaTemplate = cfg.KymaTemplate
		}, log)
		if backoff > 0 {
			return operation, backoff, err
		}
		operation = modifiedOperation
	}
	obj, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), "unable to decode kyma template", 5*time.Second, 30*time.Second, log,
			fmt.Errorf("unable to decode kyma template"))
	}

	if operation.KymaResourceNamespace == "" {
		log.Warn("namespace for Kyma resource not specified")
		return operation, 0, nil
	}
	kymaResourceName := steps.KymaName(operation)
	if kymaResourceName == "" {
		log.Info("Kyma resource name is empty, using instance.RuntimeID")

		instance, err := s.instances.GetByID(operation.InstanceID)
		if err != nil {
			log.Warn(fmt.Sprintf("Unable to get instance: %s", err.Error()))
			return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), "unable to get instance", 15*time.Second, 2*time.Minute, log, err)
		}
		kymaResourceName = steps.KymaNameFromInstance(instance)
		// save the kyma resource name if it was taken from the instance.runtimeID
		backoff := time.Duration(0)
		operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.KymaResourceName = kymaResourceName
		}, log)
		if backoff > 0 {
			return operation, backoff, nil
		}
	}
	if kymaResourceName == "" {
		log.Info("KymaResourceName is empty, skipping update Kyma resource step")
		return operation, 0, nil
	}

	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetGroupVersionKind(obj.GroupVersionKind())
	err = s.kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: operation.KymaResourceNamespace,
		Name:      kymaResourceName,
	}, kymaUnstructured)
	if err != nil {
		return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), fmt.Sprintf("unable to get Kyma Resource %s", kymaResourceName), 10*time.Second, 1*time.Minute, log, err)
	}

	log.Info(fmt.Sprintf("Updating Kyma resource: %s in namespace:%s", kymaResourceName, operation.KymaResourceNamespace))

	kymaUnstructured.SetLabels(steps.UpdatePlanLabels(kymaUnstructured.GetLabels(), operation.UpdatedPlanID))
	err = s.kcpClient.Update(context.Background(), kymaUnstructured)
	if err != nil {
		return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), fmt.Sprintf("unable to update Kyma Resource %s", kymaResourceName), 10*time.Second, 1*time.Minute, log, err)
	}

	return operation, 0, nil
}
