package update

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
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
	updateBTPRetryInterval = 5 * time.Second
	updateBTPRetryTimeout  = 2 * time.Minute
)

type SyncBTPOperatorCredentialsStep struct {
	operationManager  *process.OperationManager
	k8sClientProvider btpmanagercredentials.K8sClientProvider
}

func NewSyncBTPOperatorCredentialsStep(os storage.Operations, k8sClientProvider btpmanagercredentials.K8sClientProvider) *SyncBTPOperatorCredentialsStep {
	step := &SyncBTPOperatorCredentialsStep{
		k8sClientProvider: k8sClientProvider,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.BtpManagerDependency)
	return step
}

func (s *SyncBTPOperatorCredentialsStep) Name() string {
	return "Sync_BTP_Operator_Credentials"
}

func (s *SyncBTPOperatorCredentialsStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	clusterID := operation.InstanceDetails.ServiceManagerClusterID
	if clusterID == "" {
		clusterID = uuid.NewString()
		updatedOperation, backoff, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.InstanceDetails.ServiceManagerClusterID = clusterID
		}, log)
		if err != nil {
			log.Error("failed to update operation", "error", err)
		}
		if backoff != 0 {
			log.Error("cannot save cluster ID")
			return updatedOperation, backoff, nil
		}
		operation = updatedOperation
	}

	log.Info("syncing sap-btp-manager secret", "runtimeID", operation.RuntimeID, "clusterID", clusterID)

	k8sClient, err := s.k8sClientProvider.K8sClientForRuntimeID(operation.RuntimeID)
	if err != nil {
		log.Error("kubernetes client not available", "runtimeID", operation.RuntimeID, "error", err)
		return s.operationManager.RetryOperation(operation, "unable to get K8S client", err, updateBTPRetryInterval, updateBTPRetryTimeout, log)
	}

	current := &corev1.Secret{}
	getErr := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, current)

	if getErr != nil && !k8serrors.IsNotFound(getErr) {
		log.Warn("failed to get sap-btp-manager secret, will retry", "error", getErr)
		return s.operationManager.RetryOperation(operation, "failed to get sap-btp-manager secret", getErr, updateBTPRetryInterval, updateBTPRetryTimeout, log)
	}

	skipLabelRemoved := false
	if getErr == nil && current.Labels[btpmanagercredentials.SkipReconciliationLabel] == "true" {
		creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
		if !btpmanagercredentials.SecretStableFieldsMatch(current, creds) {
			// sm_url or tokenurl differ — user intentionally manages this secret
			log.Info("sap-btp-manager secret has skip-reconciliation label and stable fields differ — user manages this secret, skipping",
				"runtimeID", operation.RuntimeID,
			)
			return s.setDescription(operation, "BTP operator secret not updated: skip-reconciliation label present and stable fields differ — user manages this secret", log)
		}
		// sm_url and tokenurl match — label was set only for credential rotation; remove it
		log.Info("sap-btp-manager secret has skip-reconciliation label but stable fields match — removing label and syncing credentials",
			"runtimeID", operation.RuntimeID,
		)
		delete(current.Labels, btpmanagercredentials.SkipReconciliationLabel)
		if err := k8sClient.Update(context.Background(), current); err != nil {
			log.Warn("failed to remove skip-reconciliation label, will retry", "runtimeID", operation.RuntimeID, "error", err)
			return s.operationManager.RetryOperation(operation, "failed to remove skip-reconciliation label", err, updateBTPRetryInterval, updateBTPRetryTimeout, log)
		}
		skipLabelRemoved = true
	}

	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	desired, err := btpmanagercredentials.PrepareSecret(creds, clusterID)
	if err != nil {
		log.Error("credentials are incomplete, cannot build secret", "error", err)
		return s.operationManager.OperationFailed(operation, "credentials are incomplete", err, log)
	}

	if getErr == nil && btpmanagercredentials.SecretDataEqual(current, desired) {
		log.Info("sap-btp-manager secret already up to date — skipping write", "runtimeID", operation.RuntimeID)
		return s.setDescription(operation, "BTP operator secret already up to date — no write needed", log)
	}

	log.Info("writing sap-btp-manager secret to SKR cluster", "runtimeID", operation.RuntimeID, "secretExists", getErr == nil)
	if err := btpmanagercredentials.CreateOrUpdateSecret(k8sClient, desired, log); err != nil {
		log.Warn("failed to write sap-btp-manager secret, will retry", "runtimeID", operation.RuntimeID, "error", err)
		return s.operationManager.RetryOperation(operation, "failed to write sap-btp-manager secret", err, updateBTPRetryInterval, updateBTPRetryTimeout, log)
	}

	if skipLabelRemoved {
		log.Info("sap-btp-manager secret synced successfully, skip-reconciliation label removed", "runtimeID", operation.RuntimeID)
		return s.setDescription(operation, "BTP operator secret updated: skip-reconciliation label removed and credentials synced", log)
	}
	log.Info("sap-btp-manager secret synced successfully", "runtimeID", operation.RuntimeID)
	return s.setDescription(operation, "BTP operator secret updated with new credentials", log)
}

// setDescription persists a human-readable outcome description on the operation and returns it.
func (s *SyncBTPOperatorCredentialsStep) setDescription(operation internal.Operation, description string, log *slog.Logger) (internal.Operation, time.Duration, error) {
	updated, backoff, err := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.Description = description
	}, log)
	if err != nil {
		log.Error("failed to persist step description", "error", err)
	}
	return updated, backoff, nil
}
