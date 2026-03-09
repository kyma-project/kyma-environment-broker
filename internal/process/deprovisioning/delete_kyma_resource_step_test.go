package deprovisioning

import (
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteKymaResource_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage, kcpClient, fixture.FakeKymaConfigProvider{})
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.Contains(t, err.Error(), fmt.Sprintf("instance operation with id %s already exist", fixOperationID))

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestDeleteKymaResource_EmptyRuntimeIDAndKymaResourceName(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.RuntimeID = ""
	operation.KymaResourceName = ""
	instance := fixture.FixInstance(fixInstanceID)

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage, kcpClient, fixture.FakeKymaConfigProvider{})
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.Contains(t, err.Error(), fmt.Sprintf("instance operation with id %s already exist", fixOperationID))
	err = memoryStorage.Instances().Insert(instance)
	require.NoError(t, err)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}
