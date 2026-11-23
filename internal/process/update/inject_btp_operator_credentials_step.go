package update

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	btpmanagercredentials "github.com/kyma-project/kyma-environment-broker/internal/btpmanager/credentials"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	updateBTPSecretBackoff = 10 * time.Second
	updateBTPRetryInterval = 5 * time.Second
	updateBTPRetryTimeout  = 2 * time.Minute
)

type InjectBTPOperatorCredentialsStep struct {
	operationManager  *process.OperationManager
	k8sClientProvider btpmanagercredentials.K8sClientProvider
}

func NewInjectBTPOperatorCredentialsStep(os storage.Operations, k8sClientProvider btpmanagercredentials.K8sClientProvider) *InjectBTPOperatorCredentialsStep {
	step := &InjectBTPOperatorCredentialsStep{
		k8sClientProvider: k8sClientProvider,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.BtpManagerDependency)
	return step
}

func (s *InjectBTPOperatorCredentialsStep) Name() string {
	return "Update_Inject_BTP_Operator_Credentials"
}

func (s *InjectBTPOperatorCredentialsStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		log.Warn("Runtime ID is empty, retrying")
		return s.operationManager.RetryOperation(operation, "Runtime ID is empty", nil, updateBTPRetryInterval, updateBTPRetryTimeout, log)
	}

	clusterID := operation.InstanceDetails.ServiceManagerClusterID
	if clusterID == "" {
		log.Info("ServiceManagerClusterID is empty, skipping BTP operator credentials injection")
		return operation, 0, nil
	}

	k8sClient, err := s.k8sClientProvider.K8sClientForRuntimeID(operation.RuntimeID)
	if err != nil {
		log.Error(fmt.Sprintf("kubernetes client not available: %s", err))
		return s.operationManager.RetryOperation(operation, "unable to get K8S client", err, updateBTPRetryInterval, updateBTPRetryTimeout, log)
	}

	desired, err := btpmanagercredentials.PrepareSecret(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, clusterID)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "secret preparation failed", err, log)
	}

	current := &corev1.Secret{}
	getErr := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, current)
	if getErr != nil {
		if !k8serrors.IsNotFound(getErr) {
			log.Warn(fmt.Sprintf("failed to get sap-btp-manager secret: %s", getErr))
			return operation, updateBTPSecretBackoff, nil
		}
		// secret does not exist yet — create it
		if err := btpmanagercredentials.CreateOrUpdateSecret(k8sClient, desired, log); err != nil {
			err = kebError.AsTemporaryError(err, "failed create/update of the secret")
			return operation, updateBTPSecretBackoff, nil
		}
		return operation, 0, nil
	}

	if btpmanagercredentials.SecretDataEqual(current, desired) {
		log.Info("sap-btp-manager secret is up to date, skipping update")
		return operation, 0, nil
	}

	log.Info("sap-btp-manager secret differs from desired state, updating")
	if err := btpmanagercredentials.CreateOrUpdateSecret(k8sClient, desired, log); err != nil {
		err = kebError.AsTemporaryError(err, "failed create/update of the secret")
		return operation, updateBTPSecretBackoff, nil
	}
	return operation, 0, nil
}
