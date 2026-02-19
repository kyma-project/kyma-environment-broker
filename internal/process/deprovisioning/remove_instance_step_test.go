package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestRemoveInstanceStep_HappyPathForPermanentRemoval(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixDeprovisioningOperationAsOperation(testOperationID, testInstanceID)
	instance := fixture.FixInstance(testInstanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	operationFromStorage, err := memoryStorage.Operations().GetOperationByID(testOperationID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(operationFromStorage.ProvisioningParameters.ErsContext.UserID))

	_, err = memoryStorage.Instances().GetByID(testInstanceID)
	assert.ErrorContains(t, err, "not exist")

	assert.Equal(t, time.Duration(0), backoff)
}

func TestRemoveInstanceStep_UpdateOperationFailsForPermanentRemoval(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixDeprovisioningOperationAsOperation(testOperationID, testInstanceID)
	instance := fixture.FixInstance(testInstanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.Equal(t, time.Minute, backoff)
}

func TestRemoveInstanceStep_HappyPathForSuspension(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixSuspensionOperationAsOperation(testOperationID, testInstanceID)
	instance := fixture.FixInstance(testInstanceID)
	instance.DeletedAt = time.Time{}

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	operationFromStorage, err := memoryStorage.Operations().GetOperationByID(testOperationID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(operationFromStorage.RuntimeID))

	instanceFromStorage, err := memoryStorage.Instances().GetByID(testInstanceID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(instanceFromStorage.RuntimeID))
	assert.Equal(t, time.Time{}, instanceFromStorage.DeletedAt)

	assert.Equal(t, time.Duration(0), backoff)
}

func TestRemoveInstanceStep_InstanceHasExecutedButNotCompletedOperationSteps(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixDeprovisioningOperationAsOperation(testOperationID, testInstanceID)
	operation.ExcutedButNotCompleted = append(operation.ExcutedButNotCompleted, "Remove_Runtime")
	instance := fixture.FixInstance(testInstanceID)
	instance.DeletedAt = time.Time{}

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	operationFromStorage, err := memoryStorage.Operations().GetOperationByID(testOperationID)
	assert.NoError(t, err)
	assert.Equal(t, false, operationFromStorage.Temporary)

	instanceFromStorage, err := memoryStorage.Instances().GetByID(testInstanceID)
	assert.NoError(t, err)
	assert.NotEqual(t, time.Time{}, instanceFromStorage.DeletedAt)

	assert.Equal(t, time.Duration(0), backoff)
}

func TestRemoveInstanceStep_InstanceDeleted(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixDeprovisioningOperationAsOperation(testOperationID, testInstanceID)

	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewRemoveInstanceStep(memoryStorage)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.Equal(t, time.Duration(0), backoff)
}
