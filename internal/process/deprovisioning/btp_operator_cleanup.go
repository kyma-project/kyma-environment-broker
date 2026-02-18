package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors2 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorBinding         = "ServiceBinding"
)

type K8sClientProvider interface {
	K8sClientForRuntimeID(rid string) (client.Client, error)
}

type BTPOperatorCleanupStep struct {
	operationManager  *process.OperationManager
	k8sClientProvider K8sClientProvider
}

func NewBTPOperatorCleanupStep(db storage.BrokerStorage, k8sClientProvider K8sClientProvider) *BTPOperatorCleanupStep {
	step := &BTPOperatorCleanupStep{
		k8sClientProvider: k8sClientProvider,
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.BtpManagerDependency)
	return step
}

func (s *BTPOperatorCleanupStep) Name() string {
	return "BTPOperator_Cleanup"
}

func (s *BTPOperatorCleanupStep) softDelete(operation internal.Operation, k8sClient client.Client, log *slog.Logger) (internal.Operation, time.Duration, error) {
	namespaces := corev1.NamespaceList{}
	if err := k8sClient.List(context.Background(), &namespaces); err != nil {
		return s.retryOnError(operation, nil, err, log, "failed to list namespaces")
	}

	var errors []string
	gvk := schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorBinding}
	SBCrdExists, err := s.checkCRDExistence(k8sClient, gvk)
	if err != nil {
		return operation, 0, err
	}
	if SBCrdExists {
		errors = s.removeResources(k8sClient, gvk, namespaces, errors)
	}

	gvk.Kind = btpOperatorServiceInstance
	SICrdExists, err := s.checkCRDExistence(k8sClient, gvk)
	if err != nil {
		return operation, 0, err
	}
	if SICrdExists {
		errors = s.removeResources(k8sClient, gvk, namespaces, errors)
	}

	if len(errors) != 0 {
		return s.retryOnError(operation, nil, fmt.Errorf("%s", strings.Join(errors, ";")), log, "failed to cleanup")
	}
	return operation, 0, nil
}

func (s *BTPOperatorCleanupStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		log.Info("RuntimeID is empty, skipping")
		return operation, 0, nil
	}
	if !operation.Temporary {
		log.Info("skipping BTP cleanup step for real deprovisioning, not suspension")
		return operation, 0, nil
	}
	if operation.ProvisioningParameters.PlanID != broker.TrialPlanID {
		log.Info("skipping BTP cleanup step, cleanup executed only for trial plan")
		return operation, 0, nil
	}
	kclient, err := s.k8sClientProvider.K8sClientForRuntimeID(operation.RuntimeID)
	if err != nil {
		if kubeconfig.IsNotFound(err) {
			log.Info("Kubeconfig does not exists, skipping BTPOperator cleanup step")
			return operation, 0, nil
		}
		log.Warn(fmt.Sprintf("Error: %+v", err))

		return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), fmt.Sprintf("failed to get kube client: %s", err.Error()), time.Second, 30*time.Second, log, err)
	}
	if operation.UserAgent == broker.AccountCleanupJob {
		log.Info("executing soft delete cleanup for accountcleanup-job")
		return s.softDelete(operation, kclient, log)
	}
	if operation.RuntimeID == "" {
		log.Info("instance has been deprovisioned already")
		return operation, 0, nil
	}
	if kclient == nil {
		log.Info("Skipping service instance and binding deletion")
		return operation, 0, nil
	}
	if err := s.deleteServiceBindingsAndInstances(kclient, log); err != nil {
		err = kebError.AsTemporaryError(err, "failed BTP operator resource cleanup")
		return s.retryOnError(operation, kclient, err, log, "could not delete bindings and service instances")
	}
	return operation, 0, nil
}

func (s *BTPOperatorCleanupStep) deleteServiceBindingsAndInstances(k8sClient client.Client, log *slog.Logger) error {
	namespaces := corev1.NamespaceList{}
	if err := k8sClient.List(context.Background(), &namespaces); err != nil {
		return err
	}
	requeue := s.deleteResource(k8sClient, namespaces, schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorBinding}, log)
	requeue = requeue || s.deleteResource(k8sClient, namespaces, schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorServiceInstance}, log)
	if requeue {
		return fmt.Errorf("waiting for resources to be deleted")
	}
	return nil
}

func (s *BTPOperatorCleanupStep) removeFinalizers(k8sClient client.Client, namespaces corev1.NamespaceList, gvk schema.GroupVersionKind) error {
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	var errors []string
	for _, ns := range namespaces.Items {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(listGvk)
		if err := k8sClient.List(context.Background(), list, client.InNamespace(ns.Name)); err != nil {
			errors = append(errors, fmt.Sprintf("failed listing resource %v in namespace %v: %v", gvk, ns.Name, err))
		}
		for _, r := range list.Items {
			r.SetFinalizers([]string{})
			if err := k8sClient.Update(context.Background(), &r); err != nil {
				errors = append(errors, fmt.Sprintf("failed remove finalizer for resource %v %v/%v: %v", gvk, r.GetNamespace(), r.GetName(), err))
			}
		}
	}
	if len(errors) != 0 {
		return fmt.Errorf("failed to remove finalizers: %v", strings.Join(errors, ";"))
	}
	return nil
}

func (s *BTPOperatorCleanupStep) deleteResource(k8sClient client.Client, namespaces corev1.NamespaceList, gvk schema.GroupVersionKind, log *slog.Logger) (requeue bool) {
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	stillExistingCount := 0
	for _, ns := range namespaces.Items {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(listGvk)
		if err := k8sClient.List(context.Background(), list, client.InNamespace(ns.Name)); err != nil {
			log.Error(fmt.Sprintf("failed listing resource %v in namespace %v", gvk, ns.Name))
			if k8serrors2.IsNoMatchError(err) {
				// CRD doesn't exist anymore
				return false
			}
			requeue = true
		}
		stillExistingCount += len(list.Items)
	}
	if stillExistingCount == 0 {
		return
	}
	requeue = true
	for _, ns := range namespaces.Items {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		if err := k8sClient.DeleteAllOf(context.Background(), obj, client.InNamespace(ns.Name)); err != nil {
			log.Error(fmt.Sprintf("failed deleting resources %v in namespace %v", gvk, ns.Name))
		}
	}
	return
}

func (s *BTPOperatorCleanupStep) retryOnError(op internal.Operation, kclient client.Client, err error, log *slog.Logger, msg string) (internal.Operation, time.Duration, error) {
	if err != nil {
		// handleError returns retry period if it's retriable error and it's within timeout
		op, retry, err2 := handleError(s.Name(), op, err, log, msg)
		if retry != 0 {
			return op, retry, err2
		}
		// when retry is 0, that means error has been retried defined number of times and as a fallback routine
		// it was decided that KEB should try to remove finalizers once
		s.attemptToRemoveFinalizers(op, kclient, log)
		return op, retry, err2
	}
	return op, 0, nil
}

func (s *BTPOperatorCleanupStep) attemptToRemoveFinalizers(op internal.Operation, k8sClient client.Client, log *slog.Logger) {
	namespaces := corev1.NamespaceList{}
	if err := k8sClient.List(context.Background(), &namespaces); err != nil {
		log.Error(fmt.Sprintf("failed to list namespaces to remove finalizers: %v", err))
		return
	}
	if err := s.removeFinalizers(k8sClient, namespaces, schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorBinding}); err != nil {
		log.Error(fmt.Sprintf("failed to remove finalizers for bindings: %v", err))
	}
	if err := s.removeFinalizers(k8sClient, namespaces, schema.GroupVersionKind{Group: btpOperatorGroup, Version: btpOperatorApiVer, Kind: btpOperatorServiceInstance}); err != nil {
		log.Error(fmt.Sprintf("failed to remove finalizers for instances: %v", err))
	}
}

func (s *BTPOperatorCleanupStep) checkCRDExistence(k8sClient client.Client, gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: crdName}, crd); err != nil {
		if k8serrors.IsNotFound(err) || k8serrors2.IsNoMatchError(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (s *BTPOperatorCleanupStep) removeResources(k8sClient client.Client, gvk schema.GroupVersionKind, namespaces corev1.NamespaceList, errors []string) []string {
	for _, ns := range namespaces.Items {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		if err := k8sClient.DeleteAllOf(context.Background(), obj, client.InNamespace(ns.Name)); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if err := s.removeFinalizers(k8sClient, namespaces, gvk); err != nil {
		errors = append(errors, err.Error())
	}
	return errors
}
