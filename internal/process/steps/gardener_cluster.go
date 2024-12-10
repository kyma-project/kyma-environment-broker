package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const GardenerClusterStateReady = "Ready"

func NewSyncGardenerCluster(os storage.Operations, k8sClient client.Client, kimConfig broker.KimConfig) *syncGardenerCluster {
	step := &syncGardenerCluster{
		k8sClient: k8sClient,
		kimConfig: kimConfig,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func NewCheckGardenerCluster(os storage.Operations, k8sClient client.Client, kimConfig broker.KimConfig, gardenerClusterStepTimeout time.Duration) *checkGardenerCluster {
	step := &checkGardenerCluster{
		k8sClient:                  k8sClient,
		kimConfig:                  kimConfig,
		gardenerClusterStepTimeout: gardenerClusterStepTimeout,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

type checkGardenerCluster struct {
	k8sClient                  client.Client
	operationManager           *process.OperationManager
	kimConfig                  broker.KimConfig
	gardenerClusterStepTimeout time.Duration
}

func (_ *checkGardenerCluster) Name() string {
	return "Check_GardenerCluster"
}

func (s *checkGardenerCluster) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *checkGardenerCluster) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if s.kimConfig.IsDrivenByKim(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		log.Info(fmt.Sprintf("KIM is driving the process for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]))
		return operation, 0, nil
	}

	gc, err := s.GetGardenerCluster(operation.RuntimeID, operation.KymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get GardenerCluster %s/%s", operation.KymaResourceNamespace, operation.RuntimeID))
		return s.operationManager.RetryOperation(operation, "unable to get GardenerCluster", err, time.Second, 10*time.Second, log)
	}

	// check status
	state := gc.GetState()
	log.Info(fmt.Sprintf("GardenerCluster state: %s", state))
	if state != GardenerClusterStateReady {
		if time.Since(operation.UpdatedAt) > s.gardenerClusterStepTimeout {
			description := fmt.Sprintf("Waiting for GardenerCluster (%s/%s) ready state timeout.", operation.KymaResourceNamespace, operation.RuntimeID)
			log.Error(description)
			log.Info(fmt.Sprintf("GardenerCluster status: %s", gc.StatusAsString()))
			return s.operationManager.OperationFailed(operation, description, nil, log)
		}
		return operation, 500 * time.Millisecond, nil
	}
	return operation, 0, nil
}

func (s *checkGardenerCluster) GetGardenerCluster(name string, namespace string) (*GardenerCluster, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(GardenerClusterGVK())
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, obj)
	if err != nil {
		return nil, err
	}

	gc := NewGardenerClusterFromUnstructured(obj)

	return gc, nil
}

type syncGardenerCluster struct {
	k8sClient        client.Client
	kimConfig        broker.KimConfig
	operationManager *process.OperationManager
}

func (_ *syncGardenerCluster) Name() string {
	return "Sync_GardenerCluster"
}

func (s *syncGardenerCluster) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *syncGardenerCluster) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if s.kimConfig.IsDrivenByKim(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		log.Info(fmt.Sprintf("KIM is driving the process for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]))
		return operation, 0, nil
	}

	if operation.GardenerClusterName == "" {
		modifiedOperation, backoff, _ := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.GardenerClusterName = GardenerClusterName(op)
		}, log)
		if backoff != 0 {
			log.Error("cannot save the operation")
			return operation, 5 * time.Second, nil
		}
		operation = modifiedOperation
	}
	gardenerCluster, err := s.GetOrCreateNewGardenerCluster(operation.RuntimeID, operation.KymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get GardenerCluster %s/%s", operation.KymaResourceNamespace, operation.RuntimeID))
		return s.operationManager.RetryOperation(operation, "unable to get GardenerCluster", err, 3*time.Second, 20*time.Second, log)
	}
	err = gardenerCluster.SetShootName(operation.ShootName)
	if err != nil {
		log.Error(fmt.Sprintf("unable to set GardenerCluster shoot name: %s", err))
		return s.operationManager.RetryOperation(operation, "unable to set GardenerCluster shoot name", err, 3*time.Second, 20*time.Second, log)
	}
	err = gardenerCluster.SetKubecofigSecret(fmt.Sprintf("kubeconfig-%s", operation.RuntimeID), operation.KymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to set GardenerCluster kubeconfig secret: %s", err))
		return s.operationManager.RetryOperation(operation, "unable to set GardenerCluster kubeconfig secret", err, 3*time.Second, 20*time.Second, log)
	}

	obj := gardenerCluster.ToUnstructured()
	ApplyLabelsAndAnnotationsForLM(obj, operation)

	if gardenerCluster.ExistsInTheCluster() {
		err := s.k8sClient.Update(context.Background(), obj)
		if err != nil {
			log.Error(fmt.Sprintf("unable to update GardenerCluster %s/%s: %s", operation.KymaResourceNamespace, operation.RuntimeID, err.Error()))
			return s.operationManager.RetryOperation(operation, "unable to update GardenerCluster", err, 3*time.Second, 20*time.Second, log)
		}
	} else {
		err := s.k8sClient.Create(context.Background(), obj)
		if err != nil {
			log.Error(fmt.Sprintf("unable to create GardenerCluster %s/%s: %s", operation.KymaResourceNamespace, operation.RuntimeID, err.Error()))
			return s.operationManager.RetryOperation(operation, "unable to create GardenerCluster", err, 3*time.Second, 20*time.Second, log)
		}
	}

	return operation, 0, nil
}

func (s *syncGardenerCluster) GetOrCreateNewGardenerCluster(name, namespace string) (*GardenerCluster, error) {
	gardenerCluster := NewGardenerCluster(name, namespace)
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(GardenerClusterGVK())
	err := s.k8sClient.Get(context.Background(), gardenerCluster.ObjectKey(), existing)
	switch {
	case errors.IsNotFound(err):
		return gardenerCluster, nil
	case err != nil:
		return nil, err
	}
	return NewGardenerClusterFromUnstructured(existing), nil
}

func NewGardenerClusterFromUnstructured(u *unstructured.Unstructured) *GardenerCluster {
	return &GardenerCluster{obj: u}
}

func GardenerClusterGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "infrastructuremanager.kyma-project.io",
		Version: "v1",
		Kind:    "GardenerCluster",
	}
}

func GardenerClusterName(operation *internal.Operation) string {
	return strings.ToLower(operation.RuntimeID)
}

func GardenerClusterNameFromInstance(instance *internal.Instance) string {
	return strings.ToLower(instance.RuntimeID)
}

type GardenerCluster struct {
	obj *unstructured.Unstructured
}

func NewGardenerCluster(name, namespace string) *GardenerCluster {
	gardenerCluster := &unstructured.Unstructured{}
	gardenerCluster.SetGroupVersionKind(GardenerClusterGVK())
	gardenerCluster.SetName(name)
	gardenerCluster.SetNamespace(namespace)

	gardenerCluster.Object["spec"] = map[string]interface{}{
		"kubeconfig": map[string]interface{}{},
		"shoot":      map[string]interface{}{},
	}
	return &GardenerCluster{obj: gardenerCluster}
}

func (c *GardenerCluster) ObjectKey() client.ObjectKey {
	return client.ObjectKeyFromObject(c.obj)
}

func (c *GardenerCluster) SetShootName(shootName string) error {
	return unstructured.SetNestedField(c.obj.Object, shootName, "spec", "shoot", "name")
}

func (c *GardenerCluster) SetKubecofigSecret(name, namespace string) error {
	if err := unstructured.SetNestedField(c.obj.Object, name, "spec", "kubeconfig", "secret", "name"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(c.obj.Object, namespace, "spec", "kubeconfig", "secret", "namespace"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(c.obj.Object, "config", "spec", "kubeconfig", "secret", "key"); err != nil {
		return err
	}
	return nil
}

func (c *GardenerCluster) ToUnstructured() *unstructured.Unstructured {
	return c.obj
}

func (c *GardenerCluster) ToYaml() ([]byte, error) {
	return yaml.Marshal(c.obj.Object)
}

func (c *GardenerCluster) ExistsInTheCluster() bool {
	return c.obj.GetResourceVersion() != ""
}

func (c *GardenerCluster) GetState() string {
	val, found, _ := unstructured.NestedString(c.obj.Object, "status", "state")
	if found {
		return val
	}
	return ""
}

func (c *GardenerCluster) StatusAsString() string {
	st, found := c.obj.Object["status"]
	if !found {
		return "{}"
	}
	bytes, err := json.Marshal(st)
	// we do not expect errors
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

func (c *GardenerCluster) SetState(state string) error {
	return unstructured.SetNestedField(c.obj.Object, state, "status", "state")
}

func (c *GardenerCluster) SetStatusConditions(conditions interface{}) error {
	return unstructured.SetNestedField(c.obj.Object, conditions, "status", "conditions")
}
