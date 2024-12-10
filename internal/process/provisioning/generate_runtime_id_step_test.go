package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func TestNewGenerateRuntimeIDStep_LeaveRuntimeIDIfNotEmpty(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewGenerateRuntimeIDStep(memoryStorage.Operations(), memoryStorage.Instances())

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	instanceAfter, err := memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, instance.RuntimeID, instanceAfter.RuntimeID)
	assert.Equal(t, operation.RuntimeID, instanceAfter.RuntimeID)
}

func TestNewGenerateRuntimeIDStep_LeaveCreateRuntimeIDIfEmpty(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = ""
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	instance := fixInstance()
	instance.RuntimeID = ""
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	step := NewGenerateRuntimeIDStep(memoryStorage.Operations(), memoryStorage.Instances())

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	instanceAfter, err := memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, 36, len(instanceAfter.RuntimeID))
	assert.Equal(t, operation.RuntimeID, instanceAfter.RuntimeID)
}
