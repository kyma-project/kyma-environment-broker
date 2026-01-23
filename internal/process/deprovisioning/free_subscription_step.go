package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	kebErr "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

type FreeSubscriptionStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	operationStorage storage.Operations

	gardenerClient dynamic.Interface
	gardenerNS     string
}

const freeSubscriptionStepName = "Free_Subscription_Step"

var _ process.Step = &FreeSubscriptionStep{}

func NewFreeSubscriptionStep(os storage.Operations, is storage.Instances, gardenerClient dynamic.Interface, namespace string) *FreeSubscriptionStep {
	return &FreeSubscriptionStep{
		operationManager: process.NewOperationManager(os, freeSubscriptionStepName, kebErr.KEBDependency),
		instanceStorage:  is,
		operationStorage: os,
		gardenerClient:   gardenerClient,
		gardenerNS:       namespace,
	}
}

func (s *FreeSubscriptionStep) Name() string {
	return freeSubscriptionStepName
}

func (s *FreeSubscriptionStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	// The flow is:
	// - find the secret binding
	// - check if the subscription is shared or not - if yes - do nothing
	// - check if the subscription is internal or not - if yes - do nothing
	// - check if the subscription is dirty or not - if yes - do nothing
	// - if not used by other instances, free the subscription

	secretBindingName, err := s.findSecretBindingName(operation, logger)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to find the subscription secret name: %s", err.Error()))
		return s.operationManager.RetryOperation(operation, "finding the subscription secret name", err, 10*time.Second, time.Minute, logger)
	}
	if secretBindingName == "" {
		logger.Info("Subscription not assigned, nothing to release")
		return operation, 0, nil
	}
	secretBinding, err := s.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(s.gardenerNS).Get(context.Background(), secretBindingName, metav1.GetOptions{})
	if err != nil {
		msg := fmt.Sprintf("getting secret binding %s in namespace %s", secretBindingName, s.gardenerNS)
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, logger)
	}

	// check if shared
	if checkIfLabelIsTrue(secretBinding.GetLabels()["shared"]) {
		logger.Info("Subscription is shared, nothing to free")
		return operation, 0, nil
	}

	// check if internal
	if checkIfLabelIsTrue(secretBinding.GetLabels()["internal"]) {
		logger.Info("Subscription is internal, nothing to free")
		return operation, 0, nil
	}

	// check if dirty
	if checkIfLabelIsTrue(secretBinding.GetLabels()["dirty"]) {
		logger.Info("Subscription is already marked as dirty, nothing to free")
		return operation, 0, nil
	}

	shootlist, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(s.gardenerNS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		msg := fmt.Sprintf("listing Gardener shoots in namespace %s", s.gardenerNS)
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, logger)
	}

	for _, shoot := range shootlist.Items {
		sh := gardener.Shoot{Unstructured: shoot}
		if sh.GetSpecSecretBindingName() == secretBindingName {
			logger.Info(fmt.Sprintf("Subscription is still used by shoot %s, nothing to free", sh.GetName()))
			return operation, 0, nil
		}
	}

	logger.Info("Subscription is not used by any shoot, marking as dirty")
	labels := secretBinding.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["dirty"] = "true" //nolint:goconst
	secretBinding.SetLabels(labels)

	_, err = s.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(s.gardenerNS).Update(context.Background(), secretBinding, metav1.UpdateOptions{})
	if err != nil {
		msg := fmt.Sprintf("marking secret binding %s as dirty failed: %s", secretBinding.GetName(), err.Error())
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, logger)
	}
	logger.Info(fmt.Sprintf("Subscription released, secret binding name: %s", secretBinding.GetName()))

	return operation, 0, nil
}

func (s *FreeSubscriptionStep) findSecretBindingName(operation internal.Operation, logger *slog.Logger) (string, error) {
	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		return "", err
	}

	if instance.SubscriptionSecretName != "" {
		logger.Info(fmt.Sprintf("Found subscription secret name from the instance: %s", instance.SubscriptionSecretName))
		return instance.SubscriptionSecretName, nil
	}

	logger.Info("Subscription secret name not found in the instance, looking into the provisioning operation parameters")
	provisioningOp, err := s.operationStorage.GetLastOperationByTypes(operation.InstanceID, []internal.OperationType{internal.OperationTypeProvision})
	if err != nil {
		return "", err
	}

	if provisioningOp.ProvisioningParameters.Parameters.TargetSecret == nil || *provisioningOp.ProvisioningParameters.Parameters.TargetSecret == "" {
		logger.Info("instance.SubscriptionSecretName and ProvisioningParameters.TargetSecret are empty, subscription was not assigned, nothing to relese")
		return "", nil
	}
	logger.Info(fmt.Sprintf("Found subscription secret name from the provisioning operation parameters: %s", *provisioningOp.ProvisioningParameters.Parameters.TargetSecret))
	return *provisioningOp.ProvisioningParameters.Parameters.TargetSecret, nil
}
