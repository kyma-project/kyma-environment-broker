package archive

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Run(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	prepareDataForDeletedInstance(t, db, "inst-deleted-01")
	prepareDataForDeletedInstance(t, db, "inst-deleted-02")
	prepareDataForDeletedInstance(t, db, "inst-deleted-03")

	prepareDataForInstanceWithFailedDeprovisioning(t, db, "inst-failed-deprovisioning-01")
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(l)
	service := NewService(db, false, true, 10)

	// when
	numberOfInstancesProcessed, numberOfOperationsDeleted, err := service.Run()

	// then
	require.NoError(t, err)
	require.Equal(t, 3, numberOfInstancesProcessed)
	// 3 instances deprovisioned, 2operations per instance
	require.Equal(t, 6, numberOfOperationsDeleted)

	// check if operations for deleted instances are deleted
	operations, err := db.Operations().ListOperationsByInstanceID("inst-deleted-01")
	require.NoError(t, err)
	assert.Empty(t, operations)

	operations, err = db.Operations().ListOperationsByInstanceID("inst-deleted-02")
	require.NoError(t, err)
	assert.Empty(t, operations)

	operations, err = db.Operations().ListOperationsByInstanceID("inst-deleted-03")
	require.NoError(t, err)
	assert.Empty(t, operations)

	// check if operations for existing instance still exists
	operations, err = db.Operations().ListOperationsByInstanceID("inst-failed-deprovisioning-01")
	require.NoError(t, err)
	assert.NotEmpty(t, operations)

	// check if archived instances are present
	instanceArchived, err := db.InstancesArchived().GetByInstanceID("inst-deleted-01")
	require.NoError(t, err)
	assert.Equal(t, "inst-deleted-01", instanceArchived.InstanceID)

	instanceArchived, err = db.InstancesArchived().GetByInstanceID("inst-deleted-02")
	require.NoError(t, err)
	assert.Equal(t, "inst-deleted-02", instanceArchived.InstanceID)

	instanceArchived, err = db.InstancesArchived().GetByInstanceID("inst-deleted-03")
	require.NoError(t, err)
	assert.Equal(t, "inst-deleted-03", instanceArchived.InstanceID)

}

func prepareDataForDeletedInstance(t *testing.T, db storage.BrokerStorage, instanceId string) {
	provisioningOperation := fixture.FixProvisioningOperation(fmt.Sprintf("%s-%s", instanceId, "provisioninig"), instanceId)
	err := db.Operations().InsertOperation(provisioningOperation)
	require.NoError(t, err)
	err = db.Operations().InsertOperation(fixture.FixDeprovisioningOperationAsOperation(fmt.Sprintf("%s-%s", instanceId, "deprovisioning"), instanceId))
	require.NoError(t, err)
}

func prepareDataForInstanceWithFailedDeprovisioning(t *testing.T, db storage.BrokerStorage, instanceId string) {
	provisioningOperation := fixture.FixProvisioningOperation(fmt.Sprintf("%s-%s", instanceId, "provisioninig"), instanceId)
	err := db.Operations().InsertOperation(provisioningOperation)
	require.NoError(t, err)
	op := fixture.FixDeprovisioningOperationAsOperation(fmt.Sprintf("%s-%s", instanceId, "deprovisioning"), instanceId)
	op.State = domain.Failed
	err = db.Operations().InsertOperation(op)
	require.NoError(t, err)
	err = db.Instances().Insert(fixture.FixInstance(instanceId))
	require.NoError(t, err)
}
