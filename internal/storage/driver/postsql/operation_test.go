package postsql_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperation(t *testing.T) {

	t.Run("should delete operation by ID", func(t *testing.T) {
		// given
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
		op1 := fixture.FixProvisioningOperation("op-to-delete", "inst1")
		op2 := fixture.FixProvisioningOperation("op-to-keep", "inst1")

		err = brokerStorage.Operations().InsertOperation(op1)
		require.NoError(t, err)
		err = brokerStorage.Operations().InsertOperation(op2)
		require.NoError(t, err)

		// when
		err = brokerStorage.Operations().DeleteByID("op-to-delete")
		require.NoError(t, err)

		// then
		ops, err := brokerStorage.Operations().ListOperationsByInstanceID("inst1")
		require.NoError(t, err)
		assert.Equal(t, 1, len(ops))
		assert.Equal(t, "op-to-keep", ops[0].ID)
	})

	t.Run("Provisioning", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		orchestrationID := "orch-id"

		givenOperation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		givenOperation.InputCreator = nil
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = givenOperation.CreatedAt.Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.OrchestrationID = orchestrationID
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		latestOperation := fixture.FixProvisioningOperation("latest-id", "inst-id")
		latestOperation.InputCreator = nil
		latestOperation.State = domain.InProgress
		latestOperation.CreatedAt = latestOperation.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		latestOperation.UpdatedAt = latestOperation.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestOperation.Version = 1
		latestOperation.OrchestrationID = orchestrationID
		latestOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		latestPendingOperation := fixture.FixProvisioningOperation("latest-id-pending", "inst-id")
		latestPendingOperation.InputCreator = nil
		latestPendingOperation.State = orchestration.Pending
		latestPendingOperation.CreatedAt = latestPendingOperation.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestPendingOperation.UpdatedAt = latestPendingOperation.UpdatedAt.Truncate(time.Millisecond).Add(3 * time.Minute)
		latestPendingOperation.Version = 1
		latestPendingOperation.OrchestrationID = orchestrationID
		latestPendingOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		err = brokerStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: orchestrationID})
		require.NoError(t, err)

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertOperation(givenOperation)
		require.NoError(t, err)
		err = svc.InsertOperation(latestOperation)
		require.NoError(t, err)
		err = svc.InsertOperation(latestPendingOperation)
		require.NoError(t, err)

		ops, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeProvision)
		require.NoError(t, err)
		assert.Len(t, ops, 3)
		assertOperation(t, givenOperation, ops[0])

		gotOperation, err := svc.GetProvisioningOperationByID("operation-id")
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation.ID, op.ID)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, latestOperation.ID, lastOp.ID)

		lastProvisioning, err := svc.GetLastOperationByTypes("inst-id", []internal.OperationType{internal.OperationTypeProvision})
		require.NoError(t, err)
		assert.Equal(t, latestOperation.ID, lastProvisioning.ID)

		// then
		assertOperation(t, givenOperation, gotOperation.Operation)

		// when
		gotOperation.Description = "new modified description"
		_, err = svc.UpdateProvisioningOperation(*gotOperation)
		require.NoError(t, err)

		// then
		gotOperation2, err := svc.GetProvisioningOperationByID("operation-id")
		require.NoError(t, err)

		assert.Equal(t, "new modified description", gotOperation2.Description)

		// when
		stats, err := svc.GetOperationStatsByPlan()
		require.NoError(t, err)

		assert.Equal(t, 2, stats[broker.TrialPlanID].Provisioning[domain.InProgress])

		opStats, err := svc.GetOperationStatsForOrchestration(orchestrationID)
		require.NoError(t, err)

		assert.Equal(t, 2, opStats[orchestration.InProgress])

		// when
		opList, err := svc.ListProvisioningOperationsByInstanceID("inst-id")
		// then
		require.NoError(t, err)
		assert.Equal(t, 3, len(opList))
	})

	t.Run("Deprovisioning", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		givenOperation := fixture.FixDeprovisioningOperation("operation-id", "inst-id")
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = time.Now().Truncate(time.Millisecond)
		givenOperation.UpdatedAt = time.Now().Truncate(time.Millisecond).Add(time.Second)
		givenOperation.ProvisionerOperationID = "target-op-id"
		givenOperation.Description = "description"
		givenOperation.Version = 1

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertDeprovisioningOperation(givenOperation)
		require.NoError(t, err)

		ops, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeDeprovision)
		require.NoError(t, err)
		assert.Len(t, ops, 1)
		assertOperation(t, givenOperation.Operation, ops[0])

		gotOperation, err := svc.GetDeprovisioningOperationByID("operation-id")
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation.Operation.ID, op.ID)

		// then
		assertDeprovisioningOperation(t, givenOperation, *gotOperation)

		// when
		gotOperation.Description = "new modified description"
		_, err = svc.UpdateDeprovisioningOperation(*gotOperation)
		require.NoError(t, err)

		// then
		gotOperation2, err := svc.GetDeprovisioningOperationByID("operation-id")
		require.NoError(t, err)

		assert.Equal(t, "new modified description", gotOperation2.Description)

		// given
		err = svc.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         "other-op-id",
				InstanceID: "inst-id",
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
			},
		})
		require.NoError(t, err)
		// when
		opList, err := svc.ListDeprovisioningOperationsByInstanceID("inst-id")
		// then
		require.NoError(t, err)
		assert.Equal(t, 2, len(opList))
	})

	t.Run("Upgrade Cluster", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		orchestrationID := "orchestration-id"

		givenOperation1 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-1", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation1.State = domain.InProgress
		givenOperation1.CreatedAt = givenOperation1.CreatedAt.Truncate(time.Millisecond)
		givenOperation1.UpdatedAt = givenOperation1.UpdatedAt.Truncate(time.Millisecond).Add(time.Second)
		givenOperation1.ProvisionerOperationID = "target-op-id"
		givenOperation1.Description = "description"
		givenOperation1.Version = 1
		givenOperation1.OrchestrationID = orchestrationID

		givenOperation2 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-2", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation2.State = domain.InProgress
		givenOperation2.CreatedAt = givenOperation2.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		givenOperation2.UpdatedAt = givenOperation2.UpdatedAt.Truncate(time.Millisecond).Add(time.Minute).Add(time.Second)
		givenOperation2.ProvisionerOperationID = "target-op-id"
		givenOperation2.Description = "description"
		givenOperation2.Version = 1
		givenOperation2.OrchestrationID = orchestrationID
		givenOperation2.RuntimeOperation = fixRuntimeOperation("operation-id-2")

		givenOperation3 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-3", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation3.State = orchestration.Pending
		givenOperation3.CreatedAt = givenOperation3.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Hour)
		givenOperation3.UpdatedAt = givenOperation3.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute)
		givenOperation3.ProvisionerOperationID = "target-op-id"
		givenOperation3.Description = "pending-operation"
		givenOperation3.Version = 1
		givenOperation3.OrchestrationID = orchestrationID
		givenOperation3.RuntimeOperation = fixRuntimeOperation("operation-id-3")

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertUpgradeClusterOperation(givenOperation1)
		require.NoError(t, err)
		err = svc.InsertUpgradeClusterOperation(givenOperation2)
		require.NoError(t, err)
		err = svc.InsertUpgradeClusterOperation(givenOperation3)
		require.NoError(t, err)

		// then
		op, err := svc.GetUpgradeClusterOperationByID(givenOperation3.Operation.ID)
		require.NoError(t, err)
		assertUpgradeClusterOperation(t, givenOperation3, *op)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation2.Operation.ID, lastOp.ID)

		lastClusterUpgrade, err := svc.GetLastOperationByTypes("inst-id", []internal.OperationType{internal.OperationTypeUpgradeCluster})
		require.NoError(t, err)
		assert.Equal(t, givenOperation2.Operation.ID, lastClusterUpgrade.ID)

		ops, count, totalCount, err := svc.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{PageSize: 10, Page: 1})
		require.NoError(t, err)
		assert.Len(t, ops, 3)
		assert.Equal(t, count, 3)
		assert.Equal(t, totalCount, 3)

		ops, err = svc.ListUpgradeClusterOperationsByInstanceID("inst-id")
		require.NoError(t, err)
		assert.Len(t, ops, 3)

		// when
		givenOperation3.Description = "diff"
		givenOperation3.ProvisionerOperationID = "modified-op-id"
		op, err = svc.UpdateUpgradeClusterOperation(givenOperation3)
		op.CreatedAt = op.CreatedAt.Truncate(time.Millisecond)
		op.MaintenanceWindowBegin = op.MaintenanceWindowBegin.Truncate(time.Millisecond)
		op.MaintenanceWindowEnd = op.MaintenanceWindowEnd.Truncate(time.Millisecond)

		// then
		got, err := svc.GetUpgradeClusterOperationByID(givenOperation3.Operation.ID)
		require.NoError(t, err)
		assertUpgradeClusterOperation(t, *op, *got)
	})

	t.Run("Should list operations based on filters", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		globalAccountID := "dummy-ga-id"

		op1 := fixture.FixOperation("op1", "inst1", internal.OperationTypeProvision)
		op1.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		err = brokerStorage.Operations().InsertOperation(op1)
		require.NoError(t, err)

		op2 := fixture.FixOperation("op2", "inst2", internal.OperationTypeProvision)
		op2.State = domain.Failed
		op2.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		err = brokerStorage.Operations().InsertOperation(op2)
		require.NoError(t, err)

		op3 := fixture.FixOperation("op3", "inst3", internal.OperationTypeProvision)
		op3.ProvisioningParameters.PlanID = broker.FreemiumPlanID
		op3.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		err = brokerStorage.Operations().InsertOperation(op3)
		require.NoError(t, err)

		op4 := fixture.FixOperation("op4", "inst4", internal.OperationTypeDeprovision)
		op4.ProvisioningParameters.PlanID = broker.FreemiumPlanID
		err = brokerStorage.Operations().InsertOperation(op4)
		require.NoError(t, err)

		// when
		_, count, totalCount, err := brokerStorage.Operations().ListOperations(dbmodel.OperationFilter{States: []string{string(domain.Failed)}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)
	})

	t.Run("Last operation based on types", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		provisioning := fixture.FixOperation("provisioning-id", "inst-id", internal.OperationTypeProvision)
		provisioning.CreatedAt = provisioning.CreatedAt.Truncate(time.Millisecond)
		provisioning.UpdatedAt = provisioning.UpdatedAt.Truncate(time.Millisecond)

		update := fixture.FixOperation("update-id", "inst-id", internal.OperationTypeUpdate)
		update.CreatedAt = update.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		update.UpdatedAt = update.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)

		deprovisioning := fixture.FixOperation("deprovisioning-id", "inst-id", internal.OperationTypeDeprovision)
		deprovisioning.CreatedAt = deprovisioning.CreatedAt.Truncate(time.Millisecond).Add(10 * time.Minute)
		deprovisioning.UpdatedAt = deprovisioning.UpdatedAt.Truncate(time.Millisecond).Add(12 * time.Minute)

		clusterUpgrade := fixture.FixOperation("cluster-upgrade-id", "inst-id", internal.OperationTypeUpgradeCluster)
		clusterUpgrade.CreatedAt = clusterUpgrade.CreatedAt.Truncate(time.Millisecond).Add(30 * time.Minute)
		clusterUpgrade.UpdatedAt = clusterUpgrade.UpdatedAt.Truncate(time.Millisecond).Add(32 * time.Minute)

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertOperation(provisioning)
		require.NoError(t, err)
		err = svc.InsertOperation(update)
		require.NoError(t, err)
		err = svc.InsertOperation(deprovisioning)
		require.NoError(t, err)
		err = svc.InsertOperation(clusterUpgrade)
		require.NoError(t, err)

		// then
		operation, err := svc.GetLastOperationByTypes("inst-id", []internal.OperationType{
			internal.OperationTypeProvision,
			internal.OperationTypeUpdate,
			internal.OperationTypeDeprovision,
			internal.OperationTypeUpgradeCluster,
		})
		require.NoError(t, err)
		assert.Equal(t, clusterUpgrade.ID, operation.ID)

		operation, err = svc.GetLastOperationByTypes("inst-id", []internal.OperationType{
			internal.OperationTypeProvision,
			internal.OperationTypeUpdate,
			internal.OperationTypeDeprovision,
		})
		require.NoError(t, err)
		assert.Equal(t, deprovisioning.ID, operation.ID)

		operation, err = svc.GetLastOperationByTypes("inst-id", []internal.OperationType{
			internal.OperationTypeProvision,
			internal.OperationTypeUpdate,
		})
		require.NoError(t, err)
		assert.Equal(t, update.ID, operation.ID)

		operation, err = svc.GetLastOperationByTypes("inst-id", []internal.OperationType{
			internal.OperationTypeProvision,
		})
		require.NoError(t, err)
		assert.Equal(t, provisioning.ID, operation.ID)
	})
}

func assertDeprovisioningOperation(t *testing.T, expected, got internal.DeprovisioningOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStages = got.FinishedStages

	assert.Equal(t, expected, got)
}

func assertUpgradeClusterOperation(t *testing.T, expected, got internal.UpgradeClusterOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.True(t, expected.MaintenanceWindowBegin.Equal(got.MaintenanceWindowBegin))
	assert.True(t, expected.MaintenanceWindowEnd.Equal(got.MaintenanceWindowEnd))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.MaintenanceWindowBegin = got.MaintenanceWindowBegin
	expected.MaintenanceWindowEnd = got.MaintenanceWindowEnd
	expected.FinishedStages = got.FinishedStages

	assert.Equal(t, expected, got)
}

func assertOperation(t *testing.T, expected, got internal.Operation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStages = got.FinishedStages

	assert.Equal(t, expected, got)
}
