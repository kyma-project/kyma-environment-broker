package deprovisioning

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteGardenerClusterResource_HappyFlowNoObject(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kcp-system"
	operation.GardenerClusterName = "some-runtime-id"

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteGardenerClusterStep(memoryStorage.Operations(), kcpClient, memoryStorage.Instances())
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.Contains(t, err.Error(), fmt.Sprintf("instance operation with id %s already exist", fixOperationID))

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(steps.GardenerClusterGVK())
	err = kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: "kcp-system",
		Name:      "some-runtime-id",
	}, obj)
	assert.True(t, errors.IsNotFound(err), err.Error())
}

func TestDeleteGardenerClusterResource_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kcp-system"
	operation.GardenerClusterName = "some-runtime-id"

	kcpClient := fake.NewClientBuilder().
		WithRuntimeObjects(steps.NewGardenerCluster("some-runtime-id", "kcp-system").ToUnstructured()).
		Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteGardenerClusterStep(memoryStorage.Operations(), kcpClient, memoryStorage.Instances())
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.Contains(t, err.Error(), fmt.Sprintf("instance operation with id %s already exist", fixOperationID))

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(steps.GardenerClusterGVK())
	err = kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: "kcp-system",
		Name:      "some-runtime-id",
	}, obj)
	assert.True(t, errors.IsNotFound(err), err.Error())
}
