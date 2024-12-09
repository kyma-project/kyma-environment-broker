package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	provisionerAutomock "github.com/kyma-project/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestRemoveRuntimeStep_Run(t *testing.T) {
	t.Run("Should not repeat process when deprovisioning call to provisioner succeeded", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		operation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		operation.GlobalAccountID = fixGlobalAccountID
		operation.RuntimeID = fixRuntimeID
		operation.KymaResourceNamespace = "kcp-system"
		operation.KimDeprovisionsOnly = ptr.Bool(false)
		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		err = memoryStorage.Instances().Insert(fixInstanceRuntimeStatus())
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("DeprovisionRuntime", fixGlobalAccountID, fixRuntimeID).Return(fixProvisionerOperationID, nil)

		step := NewRemoveRuntimeStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, time.Minute)

		// when
		result, repeat, err := step.Run(operation.Operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.Equal(t, 0*time.Second, repeat)
		assert.Equal(t, fixProvisionerOperationID, result.ProvisionerOperationID)

		instance, err := memoryStorage.Instances().GetByID(result.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, instance.RuntimeID, fixRuntimeID)

		provisionerClient.AssertNumberOfCalls(t, "DeprovisionRuntime", 1)

	})

	t.Run("Should not call provisioner", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		operation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
		operation.GlobalAccountID = fixGlobalAccountID
		operation.RuntimeID = fixRuntimeID
		operation.KymaResourceNamespace = "kcp-system"
		operation.KimDeprovisionsOnly = ptr.Bool(true)
		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		err = memoryStorage.Instances().Insert(fixInstanceRuntimeStatus())
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("DeprovisionRuntime", fixGlobalAccountID, fixRuntimeID).Return(fixProvisionerOperationID, nil)

		step := NewRemoveRuntimeStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, time.Minute)

		// when
		result, repeat, err := step.Run(operation.Operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.Equal(t, 0*time.Second, repeat)

		instance, err := memoryStorage.Instances().GetByID(result.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, instance.RuntimeID, fixRuntimeID)

		provisionerClient.AssertNumberOfCalls(t, "DeprovisionRuntime", 0)

	})
}
