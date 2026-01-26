package deprovisioning

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const kymaResourceNamespace = "kyma-system"

func TestCheckKymaResourceDeleted_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = kymaResourceNamespace
	operation.KymaTemplate = fixture.KymaTemplate

	kcpClient := fake.NewClientBuilder().Build()

	err := fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, kymaResourceNamespace, "some-other-Runtime-ID")
	require.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckKymaResourceDeletedStep(memoryStorage, kcpClient)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
	assertNoKymaResourceWithGivenRuntimeID(t, kcpClient, operation.KymaResourceNamespace, steps.KymaName(operation))
}

func TestCheckKymaResourceDeleted_EmptyKymaResourceName(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = kymaResourceNamespace
	operation.RuntimeID = ""
	operation.KymaResourceName = ""
	operation.KymaTemplate = fixture.KymaTemplate

	kcpClient := fake.NewClientBuilder().Build()

	err := fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, kymaResourceNamespace, "some-other-Runtime-ID")
	require.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewCheckKymaResourceDeletedStep(memoryStorage, kcpClient)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
	assertNoKymaResourceWithGivenRuntimeID(t, kcpClient, operation.KymaResourceNamespace, steps.KymaName(operation))
}

func TestCheckKymaResourceDeleted_RetryWhenStillExists(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = kymaResourceNamespace
	operation.KymaTemplate = fixture.KymaTemplate

	kcpClient := fake.NewClientBuilder().Build()

	err := fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, operation.KymaResourceNamespace, operation.RuntimeID)
	require.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()
	err = memoryStorage.Operations().InsertOperation(operation)
	require.NoError(t, err)

	step := NewCheckKymaResourceDeletedStep(memoryStorage, kcpClient)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	require.NoError(t, err)
	assert.Zero(t, backoff)
}

func assertNoKymaResourceWithGivenRuntimeID(t *testing.T, kcpClient client.Client, kymaResourceNamespace string, resourceName string) {
	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "kyma",
	})
	err := kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: kymaResourceNamespace,
		Name:      resourceName,
	}, kymaUnstructured)
	assert.True(t, errors.IsNotFound(err))
}
