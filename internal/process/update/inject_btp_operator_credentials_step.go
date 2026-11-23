package update

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	btpmanagercredentials "github.com/kyma-project/kyma-environment-broker/internal/btpmanager/credentials"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
		clusterID = uuid.NewString()
		updatedOperation, backoff, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.InstanceDetails.ServiceManagerClusterID = clusterID
		}, log)
		if err != nil {
			log.Error(fmt.Sprintf("failed to update operation: %s", err))
		}
		if backoff != 0 {
			log.Error("cannot save cluster ID")
			return updatedOperation, backoff, nil
		}
		operation = updatedOperation
	}

	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials

	k8sClient, err := s.k8sClientProvider.K8sClientForRuntimeID(operation.RuntimeID)
	if err != nil {
		log.Error(fmt.Sprintf("kubernetes client not available: %s", err))
		return s.operationManager.RetryOperation(operation, "unable to get K8S client", err, updateBTPRetryInterval, updateBTPRetryTimeout, log)
	}

	current := &corev1.Secret{}
	getErr := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, current)

	if getErr != nil && !k8serrors.IsNotFound(getErr) {
		log.Warn(fmt.Sprintf("failed to get sap-btp-manager secret: %s", getErr))
		return operation, updateBTPSecretBackoff, nil
	}

	if k8serrors.IsNotFound(getErr) {
		// secret does not exist — create a fresh one with whatever non-empty fields we have
		fresh := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        btpmanagercredentials.BtpManagerSecretName,
				Namespace:   btpmanagercredentials.BtpManagerSecretNamespace,
				Labels:      btpmanagercredentials.BtpManagerLabels,
				Annotations: btpmanagercredentials.BtpManagerAnnotations,
			},
			Type: corev1.SecretTypeOpaque,
		}
		btpmanagercredentials.MergeIntoSecret(fresh, creds, clusterID)
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: btpmanagercredentials.BtpManagerSecretNamespace}}
		if err := k8sClient.Create(context.Background(), ns); err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Warn(fmt.Sprintf("failed to create %s namespace: %s", btpmanagercredentials.BtpManagerSecretNamespace, err))
			return operation, updateBTPSecretBackoff, nil
		}
		if err := k8sClient.Create(context.Background(), fresh); err != nil {
			log.Warn(fmt.Sprintf("failed to create sap-btp-manager secret: %s", err))
			return operation, updateBTPSecretBackoff, nil
		}
		return operation, 0, nil
	}

	if btpmanagercredentials.SecretDataEqual(current, buildDesiredSecret(creds, clusterID)) {
		log.Info("sap-btp-manager secret is up to date, skipping update")
		return operation, 0, nil
	}

	log.Info("sap-btp-manager secret differs from desired state, updating")
	btpmanagercredentials.MergeIntoSecret(current, creds, clusterID)
	if err := k8sClient.Update(context.Background(), current); err != nil {
		log.Warn(fmt.Sprintf("failed to update sap-btp-manager secret: %s", err))
		return operation, updateBTPSecretBackoff, nil
	}
	return operation, 0, nil
}

// buildDesiredSecret constructs a minimal Secret containing only the non-empty
// fields from creds and clusterID, used exclusively for comparison purposes.
func buildDesiredSecret(creds *internal.ServiceManagerOperatorCredentials, clusterID string) *corev1.Secret {
	s := &corev1.Secret{}
	btpmanagercredentials.MergeIntoSecret(s, creds, clusterID)
	return s
}
