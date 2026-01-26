package postsql_test

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	targetOperationID    = "target-op-id"
	descriptionModified1 = "new modified description 1"
	descriptionModified2 = "new modified description 2"
)

func TestConflict(t *testing.T) {

	t.Run("Conflict Operations", func(t *testing.T) {

		t.Run("Plain operations - provisioning", func(t *testing.T) {
			storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			defer func() {
				err := storageCleanup()
				assert.NoError(t, err)
			}()

			givenOperation := fixture.FixOperation("operation-001", "inst-id", internal.OperationTypeProvision)
			givenOperation.State = domain.InProgress
			givenOperation.ProvisionerOperationID = targetOperationID

			svc := brokerStorage.Operations()

			require.NoError(t, err)
			require.NotNil(t, brokerStorage)

			err = svc.InsertOperation(givenOperation)
			require.NoError(t, err)

			// when
			gotOperation1, err := svc.GetOperationByID("operation-001")
			require.NoError(t, err)

			gotOperation2, err := svc.GetOperationByID("operation-001")
			require.NoError(t, err)

			// when
			gotOperation1.Description = descriptionModified1
			gotOperation2.Description = descriptionModified2
			_, err = svc.UpdateOperation(*gotOperation1)
			require.NoError(t, err)

			_, err = svc.UpdateOperation(*gotOperation2)

			// then
			assertError(t, dberr.CodeConflict, err)

			// when
			err = svc.InsertOperation(*gotOperation1)

			// then
			assertError(t, dberr.CodeAlreadyExists, err)
		})

		t.Run("Plain operations - deprovisioning", func(t *testing.T) {
			storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			defer func() {
				err := storageCleanup()
				assert.NoError(t, err)
			}()

			givenOperation := fixture.FixOperation("operation-001", "inst-id", internal.OperationTypeDeprovision)
			givenOperation.State = domain.InProgress
			givenOperation.ProvisionerOperationID = targetOperationID

			svc := brokerStorage.Operations()

			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			err = svc.InsertOperation(givenOperation)
			require.NoError(t, err)

			// when
			gotOperation1, err := svc.GetOperationByID("operation-001")
			require.NoError(t, err)

			gotOperation2, err := svc.GetOperationByID("operation-001")
			require.NoError(t, err)

			// when
			gotOperation1.Description = descriptionModified1
			gotOperation2.Description = descriptionModified2
			_, err = svc.UpdateOperation(*gotOperation1)
			require.NoError(t, err)

			_, err = svc.UpdateOperation(*gotOperation2)

			// then
			assertError(t, dberr.CodeConflict, err)

			// when
			err = svc.InsertOperation(*gotOperation1)

			// then
			assertError(t, dberr.CodeAlreadyExists, err)
		})

		t.Run("Provisioning", func(t *testing.T) {
			storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			defer func() {
				err := storageCleanup()
				assert.NoError(t, err)
			}()

			givenOperation := fixture.FixProvisioningOperation("operation-001", "inst-id")
			givenOperation.State = domain.InProgress
			givenOperation.ProvisionerOperationID = targetOperationID

			svc := brokerStorage.Operations()

			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			err = svc.InsertOperation(givenOperation)
			require.NoError(t, err)

			// when
			gotOperation1, err := svc.GetProvisioningOperationByID("operation-001")
			require.NoError(t, err)

			gotOperation2, err := svc.GetProvisioningOperationByID("operation-001")
			require.NoError(t, err)

			// when
			gotOperation1.Description = descriptionModified1
			gotOperation2.Description = descriptionModified2
			_, err = svc.UpdateProvisioningOperation(*gotOperation1)
			require.NoError(t, err)

			_, err = svc.UpdateProvisioningOperation(*gotOperation2)

			// then
			assertError(t, dberr.CodeConflict, err)

			// when
			err = svc.InsertProvisioningOperation(*gotOperation1)

			// then
			assertError(t, dberr.CodeAlreadyExists, err)
		})

		t.Run("Deprovisioning", func(t *testing.T) {
			storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			defer func() {
				err := storageCleanup()
				assert.NoError(t, err)
			}()

			givenOperation := fixture.FixDeprovisioningOperation("operation-001", "inst-id")
			givenOperation.State = domain.InProgress
			givenOperation.ProvisionerOperationID = targetOperationID

			svc := brokerStorage.Deprovisioning()

			err = svc.InsertDeprovisioningOperation(givenOperation)
			require.NoError(t, err)

			// when
			gotOperation1, err := svc.GetDeprovisioningOperationByID("operation-001")
			require.NoError(t, err)

			gotOperation2, err := svc.GetDeprovisioningOperationByID("operation-001")
			require.NoError(t, err)

			// when
			gotOperation1.Description = descriptionModified1
			gotOperation2.Description = descriptionModified2
			_, err = svc.UpdateDeprovisioningOperation(*gotOperation1)
			require.NoError(t, err)

			_, err = svc.UpdateDeprovisioningOperation(*gotOperation2)

			// then
			assertError(t, dberr.CodeConflict, err)

			// when
			err = svc.InsertDeprovisioningOperation(*gotOperation1)

			// then
			assertError(t, dberr.CodeAlreadyExists, err)
		})
	})

	t.Run("Conflict Instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		svc := brokerStorage.Instances()

		inst := internal.Instance{
			InstanceID:      "abcd-01234",
			RuntimeID:       "r-id-001",
			GlobalAccountID: "ga-001",
			SubAccountID:    "sa-001",
			ServiceID:       "service-id-001",
			ServiceName:     "awesome-service",
			ServicePlanID:   "plan-id",
			ServicePlanName: "awesome-plan",
			DashboardURL:    "",
			Parameters:      internal.ProvisioningParameters{},
			ProviderRegion:  "",
			CreatedAt:       time.Now(),
			Version:         0,
		}

		err = svc.Insert(inst)
		require.NoError(t, err)

		// try an update
		inst.DashboardURL = "http://kyma.org"
		newInst, err := svc.Update(inst)
		require.NoError(t, err)

		// try another update with old version - expect conflict
		inst.DashboardURL = "---"
		_, err = svc.Update(inst)
		require.Error(t, err)
		assert.True(t, dberr.IsConflict(err))

		// try second update with correct version
		newInst.DashboardURL = "http://new.kyma.com"
		_, err = svc.Update(*newInst)
		require.NoError(t, err)
	})
}

func assertError(t *testing.T, expectedCode int, err error) {
	require.Error(t, err)

	dbe, ok := err.(dberr.Error)
	if !ok {
		assert.Fail(t, "expected DB Error Conflict")
	}
	assert.Equal(t, expectedCode, dbe.Code())
}
