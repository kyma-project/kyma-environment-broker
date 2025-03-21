package process

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeClusterOperationManager_OperationSucceeded(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewUpgradeClusterOperationManager(operations)
	op := fixUpgradeClusterOperation()
	err := operations.InsertUpgradeClusterOperation(op)
	require.NoError(t, err)

	// when
	op, when, err := opManager.OperationSucceeded(op, "task succeeded", fixLogger())

	// then
	assert.NoError(t, err)
	assert.Equal(t, domain.Succeeded, op.State)
	assert.Equal(t, time.Duration(0), when)
}

func TestUpgradeClusterOperationManager_OperationFailed(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewUpgradeClusterOperationManager(operations)
	op := fixUpgradeClusterOperation()
	err := operations.InsertUpgradeClusterOperation(op)
	require.NoError(t, err)

	errMsg := "task failed miserably"
	errOut := fmt.Errorf("error occurred")

	// when
	op, when, err := opManager.OperationFailed(op, errMsg, errOut, fixLogger())

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, "task failed miserably: error occurred")
	assert.Equal(t, domain.Failed, op.State)
	assert.Equal(t, time.Duration(0), when)

	// when
	_, _, err = opManager.OperationFailed(op, errMsg, nil, fixLogger())

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, "task failed miserably")
}

func TestUpgradeClusterOperationManager_RetryOperation(t *testing.T) {
	// given
	memory := storage.NewMemoryStorage()
	operations := memory.Operations()
	opManager := NewUpgradeClusterOperationManager(operations)
	op := internal.UpgradeClusterOperation{}
	op.UpdatedAt = time.Now()
	retryInterval := time.Hour
	errorMessage := "task failed"
	errOut := fmt.Errorf("error occurred")
	maxtime := time.Hour * 3 // allow 2 retries

	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := operations.InsertUpgradeClusterOperation(op)
	require.NoError(t, err)

	// then - first call
	op, when, err := opManager.RetryOperation(op, errorMessage, errOut, retryInterval, maxtime, fixLogger())

	// when - first retry
	assert.True(t, when > 0)
	assert.Nil(t, err)

	// then - second call
	t.Log(op.UpdatedAt.String())
	op.UpdatedAt = op.UpdatedAt.Add(-retryInterval - time.Second) // simulate wait of first retry
	t.Log(op.UpdatedAt.String())
	op, when, err = opManager.RetryOperation(op, errorMessage, errOut, retryInterval, maxtime, fixLogger())

	// when - second call => retry
	assert.True(t, when > 0)
	assert.Nil(t, err)

}

func fixUpgradeClusterOperation() internal.UpgradeClusterOperation {
	upgradeOperation := fixture.FixUpgradeClusterOperation(
		"2c538027-d1c4-41ef-a26c-c9604483cb6d",
		"2b6645a1-87e7-491d-bce3-cc0fbe16b6c0",
	)
	upgradeOperation.State = domain.InProgress

	return upgradeOperation
}
