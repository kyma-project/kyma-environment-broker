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
	"github.com/pivotal-cf/brokerapi/v12/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

type FreeCredentialsBindingStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	operationStorage storage.Operations

	gardenerClient   dynamic.Interface
	gardenerNS       string
	shootWaitTimeout time.Duration
}

const freeCredentialsBindingStepName = "Free_Credentials_Binding_Step"

func checkIfLabelIsTrue(val string) bool {
	return val == "true" //nolint:goconst
}

var _ process.Step = &FreeCredentialsBindingStep{}

func NewFreeCredentialsBindingStep(os storage.Operations, is storage.Instances, gardenerClient dynamic.Interface, namespace string) *FreeCredentialsBindingStep {
	return newFreeCredentialsBindingStepWithShootTimeout(os, is, gardenerClient, namespace, time.Hour)
}

func newFreeCredentialsBindingStepWithShootTimeout(os storage.Operations, is storage.Instances, gardenerClient dynamic.Interface, namespace string, shootWaitTimeout time.Duration) *FreeCredentialsBindingStep {
	return &FreeCredentialsBindingStep{
		operationManager: process.NewOperationManager(os, freeCredentialsBindingStepName, kebErr.KEBDependency),
		instanceStorage:  is,
		operationStorage: os,
		gardenerClient:   gardenerClient,
		gardenerNS:       namespace,
		shootWaitTimeout: shootWaitTimeout,
	}
}

func (s *FreeCredentialsBindingStep) Name() string {
	return freeCredentialsBindingStepName
}

func (s *FreeCredentialsBindingStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	// The flow is:
	// - find the credentials binding
	// - check if the subscription is shared or not - if yes - do nothing
	// - check if the subscription is internal or not - if yes - do nothing
	// - check if the subscription is dirty or not - if yes - do nothing
	// - if not used by other instances, free the subscription

	credentialsBindingName, err := s.findCredentialsBindingName(operation, logger)
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to find the subscription secret name: %s", err.Error()))
		return s.operationManager.RetryOperation(operation, "finding the subscription secret name", err, 10*time.Second, time.Minute, logger)
	}
	if credentialsBindingName == "" {
		logger.Info("Subscription not assigned, nothing to release")
		return operation, 0, nil
	}
	credentialsBinding, err := s.gardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(s.gardenerNS).Get(context.Background(), credentialsBindingName, metav1.GetOptions{})
	if err != nil {
		msg := fmt.Sprintf("getting secret binding %s in namespace %s", credentialsBindingName, s.gardenerNS)
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, logger)
	}

	// check if shared
	if checkIfLabelIsTrue(credentialsBinding.GetLabels()["shared"]) {
		logger.Info("Subscription is shared, nothing to free")
		return operation, 0, nil
	}

	// check if internal
	if checkIfLabelIsTrue(credentialsBinding.GetLabels()["internal"]) {
		logger.Info("Subscription is internal, nothing to free")
		return operation, 0, nil
	}

	// check if dirty
	if checkIfLabelIsTrue(credentialsBinding.GetLabels()["dirty"]) {
		logger.Info("Subscription is already marked as dirty, nothing to free")
		return operation, 0, nil
	}

	// Wait for the current instance's own shoot to be deleted before checking other references.
	// This handles the race where KIM is still asynchronously deleting the shoot.
	if operation.ShootName != "" {
		_, err = s.gardenerClient.Resource(gardener.ShootResource).Namespace(s.gardenerNS).Get(context.Background(), operation.ShootName, metav1.GetOptions{})
		if err == nil {
			logger.Info(fmt.Sprintf("Shoot %s for this instance still exists, waiting for deletion before releasing credentials binding %s", operation.ShootName, credentialsBindingName))
			result, backoff, retryErr := s.operationManager.RetryOperation(operation, fmt.Sprintf("shoot %s still exists, waiting for deletion", operation.ShootName), nil, 10*time.Second, s.shootWaitTimeout, logger)
			if result.State == domain.Failed {
				// Timed out waiting — mark dirty only if no other shoot references this CB.
				if s.isLastShootReferencingCB(credentialsBindingName, operation.ShootName, logger) {
					s.markCredentialsBindingDirty(credentialsBindingName, logger)
				}
			}
			return result, backoff, retryErr
		}
	}

	// Own shoot is gone — check if any other shoot still references the CB.
	shootlist, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(s.gardenerNS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		msg := fmt.Sprintf("listing Gardener shoots in namespace %s", s.gardenerNS)
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, logger)
	}
	for _, shoot := range shootlist.Items {
		sh := gardener.Shoot{Unstructured: shoot}
		if sh.GetSpecCredentialsBindingName() == credentialsBindingName {
			logger.Info(fmt.Sprintf("Credentials binding %s is still used by shoot %s, nothing to free", credentialsBindingName, sh.GetName()))
			return operation, 0, nil
		}
	}

	logger.Info(fmt.Sprintf("Credentials binding %s is not used by any shoot, marking as dirty", credentialsBindingName))
	labels := credentialsBinding.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["dirty"] = "true"
	credentialsBinding.SetLabels(labels)

	_, err = s.gardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(s.gardenerNS).Update(context.Background(), credentialsBinding, metav1.UpdateOptions{})
	if err != nil {
		msg := fmt.Sprintf("marking secret binding %s as dirty failed: %s", credentialsBinding.GetName(), err.Error())
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, logger)
	}
	logger.Info(fmt.Sprintf("Subscription released, credentialsBindingName binding name: %s", credentialsBinding.GetName()))

	return operation, 0, nil
}

func (s *FreeCredentialsBindingStep) findCredentialsBindingName(operation internal.Operation, logger *slog.Logger) (string, error) {
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

func (s *FreeCredentialsBindingStep) isLastShootReferencingCB(credentialsBindingName, ownShootName string, logger *slog.Logger) bool {
	shootlist, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(s.gardenerNS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to list shoots to check CB references: %s", err))
		return false
	}
	for _, shoot := range shootlist.Items {
		sh := gardener.Shoot{Unstructured: shoot}
		if sh.GetName() == ownShootName {
			continue
		}
		if sh.GetSpecCredentialsBindingName() == credentialsBindingName {
			return false
		}
	}
	return true
}

func (s *FreeCredentialsBindingStep) markCredentialsBindingDirty(credentialsBindingName string, logger *slog.Logger) {
	cb, err := s.gardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(s.gardenerNS).Get(context.Background(), credentialsBindingName, metav1.GetOptions{})
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to get credentials binding %s for dirty marking: %s", credentialsBindingName, err))
		return
	}
	labels := cb.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["dirty"] = "true"
	cb.SetLabels(labels)
	if _, err = s.gardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(s.gardenerNS).Update(context.Background(), cb, metav1.UpdateOptions{}); err != nil {
		logger.Warn(fmt.Sprintf("failed to mark credentials binding %s as dirty: %s", credentialsBindingName, err))
	}
}
