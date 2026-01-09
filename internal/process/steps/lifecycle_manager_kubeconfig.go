package steps

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// syncKubeconfig step ensures desired state of kubeconfig secret for lifecycle manager
type syncKubeconfig struct {
	k8sClient        client.Client
	operationManager *process.OperationManager
}

// deleteKubeconfig step ensures kubeconfig secret for lifecycle manager is removed during deprovisioning
type deleteKubeconfig struct {
	k8sClient        client.Client
	operationManager *process.OperationManager
}

func SyncKubeconfig(os storage.Operations, k8sClient client.Client) syncKubeconfig {
	step := syncKubeconfig{k8sClient: k8sClient}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func DeleteKubeconfig(os storage.Operations, k8sClient client.Client) deleteKubeconfig {
	step := deleteKubeconfig{k8sClient: k8sClient}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (_ syncKubeconfig) Name() string {
	return "Sync_Kubeconfig"
}

func (_ deleteKubeconfig) Name() string {
	return "Delete_Kubeconfig"
}

func (s syncKubeconfig) Run(o internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	secret := initSecret(o)
	if err := s.k8sClient.Create(context.Background(), secret); errors.IsAlreadyExists(err) {
		log.Info(fmt.Sprintf("Kubeconfig already exists in the secret %s, skipping", secret.Name))
	} else if err != nil {
		msg := fmt.Sprintf("failed to create kubeconfig secret %v/%v for lifecycle manager: %v", secret.Namespace, secret.Name, err)
		log.Error(msg)
		return s.operationManager.RetryOperation(o, msg, err, time.Minute, time.Minute*5, log)
	}
	return o, 0, nil
}

func (s deleteKubeconfig) Run(o internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if o.KymaResourceNamespace == "" || o.KymaResourceName == "" {
		log.Info("kubeconfig Secret should not exist, skipping")
		return o, 0, nil
	}
	secret := initSecret(o)
	if err := s.k8sClient.Delete(context.Background(), secret); err != nil && !errors.IsNotFound(err) {
		msg := fmt.Sprintf("failed to delete kubeconfig Secret %v/%v for lifecycle manager: %v", secret.Namespace, secret.Name, err)
		log.Warn(msg)
		return s.operationManager.RetryOperationWithoutFail(o, s.Name(), msg, time.Minute, time.Minute*5, log, fmt.Errorf("%s", msg))
	}
	return o, 0, nil
}

func initSecret(o internal.Operation) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.InstanceDetails.KymaResourceNamespace,
			Name:      KymaKubeconfigName(o),
		},
		StringData: map[string]string{
			"config": o.ProvisioningParameters.Parameters.Kubeconfig,
		},
	}
	ApplyLabelsAndAnnotationsForLM(secret, o)
	return secret
}
