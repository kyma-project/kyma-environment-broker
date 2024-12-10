package deprovisioning

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const GardenerClusterName = "some-gc"

func TestCheckGardenerClusterResourceDeleted_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.GardenerClusterName = GardenerClusterName

	kcpClient := fake.NewClientBuilder().
		WithRuntimeObjects(fixGardenerClusterResource(t, "kyma-system", "some-other-Runtime-ID")).
		Build()

	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckGardenerClusterDeletedStep(memoryStorage.Operations(), kcpClient)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
	assertNoGardenerClusterResourceWithGivenRuntimeID(t, kcpClient, "kyma-system", GardenerClusterName)
}

func TestCheckGardenerClusterResourceDeleted_EmptyResourceName(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.RuntimeID = ""
	operation.GardenerClusterName = ""

	kcpClient := fake.NewClientBuilder().
		WithRuntimeObjects(fixGardenerClusterResource(t, "kyma-system", "some-other-Runtime-ID")).
		Build()

	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckGardenerClusterDeletedStep(memoryStorage.Operations(), kcpClient)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	// expected: the GardenerClusterName is empty, step do nothing
	assert.Zero(t, backoff)
}

func TestCheckGardenerClusterResourceDeleted_RetryWhenStillExists(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.GardenerClusterName = GardenerClusterName

	kcpClient := fake.NewClientBuilder().
		WithRuntimeObjects(fixGardenerClusterResource(t, "kyma-system", GardenerClusterName)).
		Build()

	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckGardenerClusterDeletedStep(memoryStorage.Operations(), kcpClient)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.NotZero(t, backoff)
	assert.NoError(t, err)
}

func assertNoGardenerClusterResourceWithGivenRuntimeID(t *testing.T, kcpClient client.Client, namespace string, resourceName string) {
	gardenerClusterUnstructured := &unstructured.Unstructured{}
	gardenerClusterUnstructured.SetGroupVersionKind(steps.GardenerClusterGVK())
	err := kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      resourceName,
	}, gardenerClusterUnstructured)
	assert.True(t, errors.IsNotFound(err))
}

func fixGardenerClusterResource(t *testing.T, namespace string, name string) *unstructured.Unstructured {
	obj := fmt.Sprintf(`
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: %s
  namespace: %s
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-%s
      namespace: kcp-system`, name, namespace, name)
	scheme := internal.NewSchemeForTests(t)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	unstructuredObject := &unstructured.Unstructured{}
	_, _, err := decoder.Decode([]byte(obj), nil, unstructuredObject)
	assert.NoError(t, err)
	return unstructuredObject
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("testing", true)
}
