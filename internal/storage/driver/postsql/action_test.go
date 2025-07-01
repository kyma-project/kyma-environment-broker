package postsql_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAction(t *testing.T) {
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
	require.NoError(t, err)
	require.NotNil(t, brokerStorage)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()
	instanceID := "instance-id"
	err = brokerStorage.Instances().Insert(fixture.FixInstance(instanceID))
	require.NoError(t, err)

	actions, err := brokerStorage.Actions().ListActionsByInstanceID(instanceID)
	assert.NoError(t, err)
	assert.Len(t, actions, 0)

	err = brokerStorage.Actions().InsertAction(internal.PlanUpdateActionType, instanceID, "test-message-1", "old-value-1", "new-value-1")
	assert.NoError(t, err)
	err = brokerStorage.Actions().InsertAction(internal.SubaccountMovementActionType, instanceID, "test-message-2", "old-value-2", "new-value-2")
	assert.NoError(t, err)

	actions, err = brokerStorage.Actions().ListActionsByInstanceID(instanceID)
	assert.NoError(t, err)
	assert.Len(t, actions, 2)

	assert.NotEmpty(t, actions[0].ID)
	assert.Equal(t, actions[0].Type, internal.SubaccountMovementActionType)
	assert.Equal(t, actions[0].InstanceID, instanceID)
	assert.Equal(t, actions[0].Message, "test-message-2")
	assert.Equal(t, actions[0].OldValue, "old-value-2")
	assert.Equal(t, actions[0].NewValue, "new-value-2")
	assert.NotEmpty(t, actions[0].CreatedAt)

	assert.NotEmpty(t, actions[1].ID)
	assert.Equal(t, actions[1].Type, internal.PlanUpdateActionType)
	assert.Equal(t, actions[1].InstanceID, instanceID)
	assert.Equal(t, actions[1].Message, "test-message-1")
	assert.Equal(t, actions[1].OldValue, "old-value-1")
	assert.Equal(t, actions[1].NewValue, "new-value-1")
	assert.NotEmpty(t, actions[1].CreatedAt)

	err = brokerStorage.InstancesArchived().Insert(fixInstanceArchive(instanceArchiveData{InstanceID: instanceID}))
	assert.NoError(t, err)

	err = brokerStorage.Instances().Delete(instanceID)
	assert.NoError(t, err)

	actions, err = brokerStorage.Actions().ListActionsByInstanceID(instanceID)
	assert.NoError(t, err)
	assert.Len(t, actions, 2)
}
