package broker_test

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
)

const (
	operationID          = "23caac24-c317-47d0-bd2f-6b1bf4bdba99"
	operationDescription = "some operation status description"
	instID               = "c39d9b98-5ed9-4a68-b786-f26ce93a734f"
)

func TestLastOperation_LastOperation(t *testing.T) {
	t.Run("Should return last operation when operation ID provided", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertOperation(fixOperation())
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: operationID})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: operationDescription,
		}, response)
	})
	t.Run("Should return last operation when operation ID not provided", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertOperation(fixOperation())
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: operationDescription,
		}, response)
	})
	t.Run("Should convert operation's pending state to in progress", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		updateOp := fixture.FixUpdatingOperation(operationID, instID)
		updateOp.State = internal.OperationStatePending
		err := memoryStorage.Operations().InsertUpdatingOperation(updateOp)
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.Error(t, err, "instance operation with instance_id %s not found", instID)

		// then
		assert.Equal(t, domain.LastOperation{}, response)

		// when
		response, err = lastOperationEndpoint.LastOperation(context.TODO(), instID,
			domain.PollDetails{OperationData: operationID})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.InProgress,
			Description: updateOp.Description,
		}, response)
	})
	t.Run("Should convert operation's retrying state to in progress", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		updateOp := fixture.FixUpdatingOperation(operationID, instID)
		updateOp.State = internal.OperationStateRetrying
		err := memoryStorage.Operations().InsertUpdatingOperation(updateOp)
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.InProgress,
			Description: updateOp.Description,
		}, response)

		// when
		response, err = lastOperationEndpoint.LastOperation(context.TODO(), instID,
			domain.PollDetails{OperationData: operationID})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.InProgress,
			Description: updateOp.Description,
		}, response)
	})
	t.Run("Should convert operation's canceling state to succeeded", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		updateOp := fixture.FixUpdatingOperation(operationID, instID)
		updateOp.State = internal.OperationStateCanceling
		err := memoryStorage.Operations().InsertUpdatingOperation(updateOp)
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: updateOp.Description,
		}, response)

		// when
		response, err = lastOperationEndpoint.LastOperation(context.TODO(), instID,
			domain.PollDetails{OperationData: operationID})
		assert.NoError(t, err)
		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: updateOp.Description,
		}, response)
	})
	t.Run("Should convert operation's canceled state to succeeded", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		updateOp := fixture.FixUpdatingOperation(operationID, instID)
		updateOp.State = internal.OperationStateCanceled
		err := memoryStorage.Operations().InsertUpdatingOperation(updateOp)
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: updateOp.Description,
		}, response)

		// when
		response, err = lastOperationEndpoint.LastOperation(context.TODO(), instID,
			domain.PollDetails{OperationData: operationID})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: updateOp.Description,
		}, response)
	})
	t.Run("Should return provisioning operation", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		provisioning := fixOperation()
		provisioning.ID = "provisioning-id"
		provisioning.Description = "Provisioning description"
		provisioning.CreatedAt = provisioning.CreatedAt.Truncate(time.Millisecond)
		provisioning.UpdatedAt = provisioning.UpdatedAt.Truncate(time.Millisecond)
		err := memoryStorage.Operations().InsertOperation(provisioning)
		assert.NoError(t, err)

		lastOperationEndpoint := broker.NewLastOperation(memoryStorage.Operations(), memoryStorage.InstancesArchived(), fixLogger())

		// when
		response, err := lastOperationEndpoint.LastOperation(context.TODO(), instID, domain.PollDetails{OperationData: ""})
		assert.NoError(t, err)

		// then
		assert.Equal(t, domain.LastOperation{
			State:       domain.Succeeded,
			Description: "Provisioning description",
		}, response)
	})
}

func fixOperation() internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperation(operationID, instID)
	provisioningOperation.State = domain.Succeeded
	provisioningOperation.Description = operationDescription

	return provisioningOperation
}
