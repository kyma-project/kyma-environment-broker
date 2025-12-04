package postsql_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperation(t *testing.T) {

	cfg := brokerStorageDatabaseTestConfig()

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

	t.Run("Provisioning in Shanghai", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTestUsingConnectionURL(cfg,
			fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode, "'Asia/Shanghai'"))
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		givenOperation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = time.Now().Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID
		givenOperation.RuntimeOperation.Region = fixture.Region
		givenOperation.RuntimeOperation.GlobalAccountID = fixture.GlobalAccountId

		svc := brokerStorage.Operations()

		timeZones := brokerStorage.TimeZones()
		tz, err := timeZones.GetTimeZone()
		require.NoError(t, err)
		require.NotEmpty(t, tz)

		// trim surrounding quotes if present
		locName := strings.Trim(tz, "'\"")
		loc, err := time.LoadLocation(locName)
		require.NoError(t, err)
		_, offsetSeconds := time.Now().In(loc).Zone()
		offset := time.Duration(offsetSeconds) * time.Second

		// when
		err = svc.InsertOperation(givenOperation)
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		require.Equal(t, givenOperation.CreatedAt.Sub(op.CreatedAt), offset)

		op, err = svc.UpdateOperation(*op)
		require.NoError(t, err)

		op, err = svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		require.Equal(t, givenOperation.CreatedAt.Sub(op.CreatedAt), 2*offset)

		op, err = svc.UpdateOperation(*op)
		require.NoError(t, err)

		op, err = svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		require.Equal(t, givenOperation.CreatedAt.Sub(op.CreatedAt), 3*offset)

		assert.Equal(t, givenOperation.ID, op.ID)
	})

	t.Run("Provisioning in UTC", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		givenOperation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = time.Now().Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID
		givenOperation.RuntimeOperation.Region = fixture.Region
		givenOperation.RuntimeOperation.GlobalAccountID = fixture.GlobalAccountId

		svc := brokerStorage.Operations()

		timeZones := brokerStorage.TimeZones()
		tz, err := timeZones.GetTimeZone()
		require.NoError(t, err)
		require.Equal(t, "UTC", tz)

		// when
		err = svc.InsertOperation(givenOperation)
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		//log the difference
		require.True(t, givenOperation.CreatedAt.Equal(op.CreatedAt))
		require.NoError(t, err)

		op, err = svc.UpdateOperation(*op)
		require.NoError(t, err)

		op, err = svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		require.True(t, givenOperation.CreatedAt.Equal(op.CreatedAt))

		op, err = svc.UpdateOperation(*op)
		require.NoError(t, err)

		op, err = svc.GetOperationByID("operation-id")
		require.True(t, givenOperation.CreatedAt.Equal(op.CreatedAt))
		require.NoError(t, err)

		assert.Equal(t, givenOperation.ID, op.ID)
	})

	t.Run("Provisioning in Los Angeles", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTestUsingConnectionURL(cfg,
			fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode, "'America/Los_Angeles'"))
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		givenOperation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = time.Now().Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID
		givenOperation.RuntimeOperation.Region = fixture.Region
		givenOperation.RuntimeOperation.GlobalAccountID = fixture.GlobalAccountId

		svc := brokerStorage.Operations()

		timeZones := brokerStorage.TimeZones()
		tz, err := timeZones.GetTimeZone()
		require.NoError(t, err)
		require.NotEmpty(t, tz)
		// trim surrounding quotes if present
		locName := strings.Trim(tz, "'\"")
		loc, err := time.LoadLocation(locName)
		require.NoError(t, err)
		_, offsetSeconds := time.Now().In(loc).Zone()
		offset := time.Duration(offsetSeconds) * time.Second

		// when
		err = svc.InsertOperation(givenOperation)
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation.ID, op.ID)
		require.Equal(t, givenOperation.CreatedAt.Sub(op.CreatedAt), offset)

		op, err = svc.UpdateOperation(*op)
		require.NoError(t, err)

		op, err = svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		require.Equal(t, givenOperation.CreatedAt.Sub(op.CreatedAt), 2*offset)

		op, err = svc.UpdateOperation(*op)
		require.NoError(t, err)

		op, err = svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		require.Equal(t, givenOperation.CreatedAt.Sub(op.CreatedAt), 3*offset)

		assert.Equal(t, givenOperation.ID, op.ID)
	})

	t.Run("Provisioning", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		givenOperation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = givenOperation.CreatedAt.Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID
		givenOperation.RuntimeOperation.Region = fixture.Region
		givenOperation.RuntimeOperation.GlobalAccountID = fixture.GlobalAccountId

		latestOperation := fixture.FixProvisioningOperation("latest-id", "inst-id")
		latestOperation.State = domain.InProgress
		latestOperation.CreatedAt = latestOperation.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		latestOperation.UpdatedAt = latestOperation.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestOperation.Version = 1
		latestOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		latestPendingOperation := fixture.FixProvisioningOperation("latest-id-pending", "inst-id")
		latestPendingOperation.State = internal.OperationStatePending
		latestPendingOperation.CreatedAt = latestPendingOperation.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestPendingOperation.UpdatedAt = latestPendingOperation.UpdatedAt.Truncate(time.Millisecond).Add(3 * time.Minute)
		latestPendingOperation.Version = 1
		latestPendingOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

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
		assertRuntimeOperation(t, givenOperation)

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
		givenOperation.RuntimeOperation.Region = fixture.Region
		givenOperation.RuntimeOperation.GlobalAccountID = fixture.GlobalAccountId

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertDeprovisioningOperation(givenOperation)
		require.NoError(t, err)

		ops, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeDeprovision)
		require.NoError(t, err)
		assert.Len(t, ops, 1)
		assertOperation(t, givenOperation.Operation, ops[0])
		assertRuntimeOperation(t, ops[0])

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

		givenOperation1 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-1", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation1.State = domain.InProgress
		givenOperation1.CreatedAt = givenOperation1.CreatedAt.Truncate(time.Millisecond)
		givenOperation1.UpdatedAt = givenOperation1.UpdatedAt.Truncate(time.Millisecond).Add(time.Second)
		givenOperation1.ProvisionerOperationID = "target-op-id"
		givenOperation1.Description = "description"
		givenOperation1.Version = 1

		givenOperation2 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-2", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation2.State = domain.InProgress
		givenOperation2.CreatedAt = givenOperation2.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		givenOperation2.UpdatedAt = givenOperation2.UpdatedAt.Truncate(time.Millisecond).Add(time.Minute).Add(time.Second)
		givenOperation2.ProvisionerOperationID = "target-op-id"
		givenOperation2.Description = "description"
		givenOperation2.Version = 1

		givenOperation3 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-3", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation3.State = internal.OperationStatePending
		givenOperation3.CreatedAt = givenOperation3.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Hour)
		givenOperation3.UpdatedAt = givenOperation3.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute)
		givenOperation3.ProvisionerOperationID = "target-op-id"
		givenOperation3.Description = "pending-operation"
		givenOperation3.Version = 1
		givenOperation3.RuntimeOperation.Region = fixture.Region
		givenOperation3.RuntimeOperation.GlobalAccountID = fixture.GlobalAccountId

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
		assertRuntimeOperation(t, op.Operation)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation2.Operation.ID, lastOp.ID)

		lastClusterUpgrade, err := svc.GetLastOperationByTypes("inst-id", []internal.OperationType{internal.OperationTypeUpgradeCluster})
		require.NoError(t, err)
		assert.Equal(t, givenOperation2.Operation.ID, lastClusterUpgrade.ID)

		ops, err := svc.ListUpgradeClusterOperationsByInstanceID("inst-id")
		require.NoError(t, err)
		assert.Len(t, ops, 3)

		// when
		givenOperation3.Description = "diff"
		givenOperation3.ProvisionerOperationID = "modified-op-id"
		op, err = svc.UpdateUpgradeClusterOperation(givenOperation3)
		op.CreatedAt = op.CreatedAt.Truncate(time.Millisecond)

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

func TestOperation_ModeCFB(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ProvisioningParameters.ErsContext = internal.ERSContext{
		SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
			ClientID:     "sm-client-id",
			ClientSecret: "sm-client-secret",
		},
	}
	operation.ProvisioningParameters.Parameters.Kubeconfig = "kube-config-data"

	// when
	err = brokerStorage.Operations().InsertOperation(operation)
	require.NoError(t, err)

	// when
	retrievedOperation, err := brokerStorage.Operations().GetOperationByID("op-id")
	require.NoError(t, err)

	statsForOperations, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForOperations()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeCFB: 1}, statsForOperations))

	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret, retrievedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID, retrievedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, operation.ProvisioningParameters.Parameters.Kubeconfig, retrievedOperation.ProvisioningParameters.Parameters.Kubeconfig)
}

func TestOperation_ModeGCM(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", true)
	encrypter.SetWriteGCMMode(true)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ProvisioningParameters.ErsContext = internal.ERSContext{
		SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
			ClientID:     "sm-client-id",
			ClientSecret: "sm-client-secret",
		},
	}
	operation.ProvisioningParameters.Parameters.Kubeconfig = "kube-config-data"

	// when
	err = brokerStorage.Operations().InsertOperation(operation)
	require.NoError(t, err)

	statsForOperations, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForOperations()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeGCM: 1}, statsForOperations))

	// when
	retrievedOperation, err := brokerStorage.Operations().GetOperationByID("op-id")
	require.NoError(t, err)

	// then
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret, retrievedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID, retrievedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID)
	// assert kubeconfig
	assert.Equal(t, operation.ProvisioningParameters.Parameters.Kubeconfig, retrievedOperation.ProvisioningParameters.Parameters.Kubeconfig)

}

func TestOperation_BothModes(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ProvisioningParameters.ErsContext = internal.ERSContext{
		SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
			ClientID:     "sm-client-id",
			ClientSecret: "sm-client-secret",
		},
	}
	operation.ProvisioningParameters.Parameters.Kubeconfig = "kube-config-data"

	// when
	err = brokerStorage.Operations().InsertOperation(operation)
	require.NoError(t, err)

	statsForOperations, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForOperations()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeCFB: 1}, statsForOperations))

	// when
	encrypter.SetWriteGCMMode(true)

	operationGCM := fixture.FixProvisioningOperation("op-id-gcm", "inst-id-gcm")
	operationGCM.ProvisioningParameters.ErsContext = internal.ERSContext{
		SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
			ClientID:     "sm-client-id-gcm",
			ClientSecret: "sm-client-secret-gcm",
		},
	}
	operationGCM.ProvisioningParameters.Parameters.Kubeconfig = "kube-config-data-gcm"

	// when
	err = brokerStorage.Operations().InsertOperation(operationGCM)
	require.NoError(t, err)

	// when
	statsForOperations, err = brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForOperations()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeCFB: 1, storage.EncryptionModeGCM: 1}, statsForOperations))

	// when
	retrievedOperation, err := brokerStorage.Operations().GetOperationByID("op-id")
	require.NoError(t, err)

	retrievedOperationGCM, err := brokerStorage.Operations().GetOperationByID("op-id-gcm")
	require.NoError(t, err)

	// then
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret, retrievedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID, retrievedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID)
	// assert kubeconfig
	assert.Equal(t, operation.ProvisioningParameters.Parameters.Kubeconfig, retrievedOperation.ProvisioningParameters.Parameters.Kubeconfig)

	assert.Equal(t, operationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret, retrievedOperationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, operationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID, retrievedOperationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID)
	// assert kubeconfig
	assert.Equal(t, operationGCM.ProvisioningParameters.Parameters.Kubeconfig, retrievedOperationGCM.ProvisioningParameters.Parameters.Kubeconfig)

	// then we update both operations and verify we can still read the secrets
	retrievedOperation.Description = "updated description"
	_, err = brokerStorage.Operations().UpdateOperation(*retrievedOperation)
	require.NoError(t, err)

	updatedOperation, err := brokerStorage.Operations().GetOperationByID("op-id")
	require.NoError(t, err)
	assert.Equal(t, "updated description", updatedOperation.Description)
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret, updatedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID, updatedOperation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID)

	retrievedOperationGCM.Description = "updated description gcm"
	_, err = brokerStorage.Operations().UpdateOperation(*retrievedOperationGCM)
	require.NoError(t, err)
	updatedOperationGCM, err := brokerStorage.Operations().GetOperationByID("op-id-gcm")
	require.NoError(t, err)
	assert.Equal(t, "updated description gcm", updatedOperationGCM.Description)
	assert.Equal(t, operationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret, updatedOperationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, operationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID, updatedOperationGCM.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, operationGCM.ProvisioningParameters.Parameters.Kubeconfig, updatedOperationGCM.ProvisioningParameters.Parameters.Kubeconfig)

	// check stats again
	statsForOperations, err = brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForOperations()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeGCM: 2}, statsForOperations))
}

func assertRuntimeOperation(t *testing.T, operation internal.Operation) {
	assert.Equal(t, fixture.GlobalAccountId, operation.RuntimeOperation.GlobalAccountID)
	assert.Equal(t, fixture.Region, operation.RuntimeOperation.Region)
}

func assertDeprovisioningOperation(t *testing.T, expected, got internal.DeprovisioningOperation) {
	// do not check zones and monotonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStages = got.FinishedStages

	assert.Equal(t, expected, got)
}

func assertUpgradeClusterOperation(t *testing.T, expected, got internal.UpgradeClusterOperation) {
	// do not check zones and monotonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStages = got.FinishedStages

	assert.Equal(t, expected, got)
}

func assertOperation(t *testing.T, expected, got internal.Operation) {
	// do not check zones and monotonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStages = got.FinishedStages

	assert.Equal(t, expected, got)
}
