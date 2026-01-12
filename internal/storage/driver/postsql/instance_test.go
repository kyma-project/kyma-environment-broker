package postsql_test

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/predicate"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstance_UsingLastOperationID(t *testing.T) {
	cfg := brokerStorageDatabaseTestConfig()

	t.Run("Should create and update instance", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		testInstanceId := "test"
		expiredID := "expired-id"
		fixInstance := fixture.FixInstance(testInstanceId)
		expiredInstance := fixture.FixInstance(expiredID)
		expiredInstance.ExpiredAt = ptr.Time(time.Now())

		err = brokerStorage.Instances().Insert(fixInstance)
		require.NoError(t, err)
		err = brokerStorage.Instances().Insert(expiredInstance)
		require.NoError(t, err)

		fixInstance.DashboardURL = "diff"
		fixInstance.Provider = pkg.SapConvergedCloud
		_, err = brokerStorage.Instances().Update(fixInstance)
		require.NoError(t, err)

		fixProvisioningOperation1 := fixture.FixProvisioningOperation("op-id", fixInstance.InstanceID)

		err = brokerStorage.Operations().InsertOperation(fixProvisioningOperation1)
		require.NoError(t, err)

		fixProvisioningOperation2 := fixture.FixProvisioningOperation("latest-op-id", fixInstance.InstanceID)

		err = brokerStorage.Operations().InsertOperation(fixProvisioningOperation2)
		require.NoError(t, err)

		// then
		inst, err := brokerStorage.Instances().GetByID(testInstanceId)
		assert.NoError(t, err)
		expired, err := brokerStorage.Instances().GetByID(expiredID)
		assert.NoError(t, err)
		require.NotNil(t, inst)

		assert.Equal(t, fixInstance.InstanceID, inst.InstanceID)
		assert.Equal(t, fixInstance.RuntimeID, inst.RuntimeID)
		assert.Equal(t, fixInstance.GlobalAccountID, inst.GlobalAccountID)
		assert.Equal(t, fixInstance.SubscriptionGlobalAccountID, inst.SubscriptionGlobalAccountID)
		assert.Equal(t, fixInstance.ServiceID, inst.ServiceID)
		assert.Equal(t, fixInstance.ServicePlanID, inst.ServicePlanID)
		assert.Equal(t, fixInstance.SubscriptionSecretName, inst.SubscriptionSecretName)
		assert.Equal(t, fixInstance.DashboardURL, inst.DashboardURL)
		assert.Equal(t, fixInstance.Parameters, inst.Parameters)
		assert.Equal(t, fixInstance.Provider, inst.Provider)
		assert.False(t, inst.IsExpired())
		assert.NotEmpty(t, inst.CreatedAt)
		assert.NotEmpty(t, inst.UpdatedAt)
		assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", inst.DeletedAt.String())
		assert.True(t, expired.IsExpired())

		// insert event
		events.Infof(fixInstance.InstanceID, fixProvisioningOperation2.ID, "some event")
		events.Errorf(fixInstance.InstanceID, fixProvisioningOperation2.ID, fmt.Errorf(""), "asdasd %s", "")

		// when
		err = brokerStorage.Instances().Delete(fixInstance.InstanceID)

		// then
		assert.NoError(t, err)
		_, err = brokerStorage.Instances().GetByID(fixInstance.InstanceID)
		assert.True(t, dberr.IsNotFound(err))

		// when
		err = brokerStorage.Instances().Delete(fixInstance.InstanceID)
		assert.NoError(t, err, "deletion non existing instance must not cause any error")
	})

	t.Run("Should fetch instance statistics", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "A1", globalAccountID: "A", subAccountID: "sub-01"}),
			*fixInstance(instanceData{val: "A2", globalAccountID: "A", subAccountID: "sub-02", deletedAt: time.Time{}}),
			*fixInstance(instanceData{val: "A3", globalAccountID: "A", subAccountID: "sub-02"}),
			*fixInstance(instanceData{val: "C1", globalAccountID: "C", subAccountID: "sub-01"}),
			*fixInstance(instanceData{val: "C2", globalAccountID: "C", deletedAt: time.Now()}),
			*fixInstance(instanceData{val: "B1", globalAccountID: "B", deletedAt: time.Now()}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		fixOperations := []internal.Operation{
			fixture.FixProvisioningOperation("op1", "A1"),
			fixture.FixProvisioningOperation("op2", "A2"),
			fixture.FixSuspensionOperationAsOperation("op3", "A3"),
			fixture.FixProvisioningOperation("op4", "C1"),
			fixture.FixProvisioningOperation("op5", "C2"),
			fixture.FixProvisioningOperation("op6", "B1"),
		}

		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertOperation(i)
			require.NoError(t, err)
			err = brokerStorage.Instances().UpdateInstanceLastOperation(i.InstanceID, i.ID)
			require.NoError(t, err)
		}

		// when
		stats, err := brokerStorage.Instances().GetActiveInstanceStats()
		require.NoError(t, err)
		numberOfInstancesA, err := brokerStorage.Instances().GetNumberOfInstancesForGlobalAccountID("A")
		require.NoError(t, err)
		numberOfInstancesC, err := brokerStorage.Instances().GetNumberOfInstancesForGlobalAccountID("C")
		require.NoError(t, err)
		numberOfInstancesB, err := brokerStorage.Instances().GetNumberOfInstancesForGlobalAccountID("B")
		require.NoError(t, err)

		t.Logf("%+v", stats)

		// then
		assert.Equal(t, internal.InstanceStats{
			TotalNumberOfInstances: 3,
			PerGlobalAccountID:     map[string]int{"A": 2, "C": 1},
			PerSubAcocuntID:        map[string]int{"sub-01": 2},
		}, stats)
		assert.Equal(t, 3, numberOfInstancesA)
		assert.Equal(t, 1, numberOfInstancesC)
		assert.Equal(t, 0, numberOfInstancesB)
	})

	t.Run("Should fetch ERS context statistics", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "A1", globalAccountID: "A", subAccountID: "sub-01"}),
			*fixInstance(instanceData{val: "A2", globalAccountID: "A", subAccountID: "sub-02", deletedAt: time.Time{}}),
			*fixInstance(instanceData{val: "A3", globalAccountID: "A", subAccountID: "sub-02"}),
			*fixInstance(instanceData{val: "C1", globalAccountID: "C", subAccountID: "sub-01"}),
			*fixInstance(instanceData{val: "C2", globalAccountID: "C", deletedAt: time.Now()}), // deleted - should not be counted
			*fixInstance(instanceData{val: "B1", globalAccountID: "B", deletedAt: time.Now()}), // deleted - should not be counted
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		op1 := fixture.FixProvisioningOperation("op1", "A1")

		op2 := fixture.FixProvisioningOperation("op2", "A2")
		op2.ProvisioningParameters.ErsContext.LicenseType = ptr.String("SAPOTHER")

		op3 := fixture.FixSuspensionOperationAsOperation("op3", "A3")

		// simulating update with different license type, since query is based on created_at date we need to adjust creation times
		// provisioning precedes update
		op4 := fixture.FixProvisioningOperation("op4", "C1")
		op4.CreatedAt = time.Date(2025, 2, 19, 11, 0, 0, 0, time.UTC)
		op5 := fixture.FixUpdatingOperation("op5", "C1").Operation
		op5.CreatedAt = time.Date(2025, 2, 19, 12, 0, 0, 0, time.UTC)
		op5.ProvisioningParameters.ErsContext.LicenseType = ptr.String("SAPOTHER")

		op6 := fixture.FixProvisioningOperation("op6", "C2") // this instance is deleted, should not be counted

		// simulating update with different license type, since query is based on created_at date we need to adjust creation times
		// but the instance is already deleted
		op7 := fixture.FixProvisioningOperation("op7", "B1")
		op7.CreatedAt = time.Date(2025, 2, 19, 11, 0, 0, 0, time.UTC)
		op8 := fixture.FixUpdatingOperation("op8", "B1").Operation
		op8.CreatedAt = time.Date(2025, 2, 19, 12, 0, 0, 0, time.UTC)
		op8.ProvisioningParameters.ErsContext.LicenseType = ptr.String("SAPOTHER")

		fixOperations := []internal.Operation{op1, op2, op3, op4, op5, op6, op7, op8}

		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertOperation(i)
			require.NoError(t, err)
			err = brokerStorage.Instances().UpdateInstanceLastOperation(i.InstanceID, i.ID)
			require.NoError(t, err)
		}

		// when
		stats, err := brokerStorage.Instances().GetERSContextStats()
		require.NoError(t, err)

		t.Logf("%+v", stats)

		// then
		assert.Equal(t, internal.ERSContextStats{LicenseType: map[string]int{"SAPDEV": 2, "SAPOTHER": 2}}, stats)
	})

	t.Run("Should get distinct subaccounts from active instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "A1", globalAccountID: "ga1", subAccountID: "sa1", runtimeID: "runtimeID1"}),
			*fixInstance(instanceData{val: "A2", globalAccountID: "ga1", subAccountID: "sa1", runtimeID: "runtimeID2"}),
			*fixInstance(instanceData{val: "A3", globalAccountID: "ga1", subAccountID: "sa2", runtimeID: "runtimeID3"}),
			*fixInstance(instanceData{val: "A4", globalAccountID: "ga2", subAccountID: "sa3", runtimeID: "runtimeID4"}),
			*fixInstance(instanceData{val: "A5", globalAccountID: "ga2", subAccountID: "sa4", runtimeID: "runtimeID5"}),
			*fixInstance(instanceData{val: "A6", globalAccountID: "ga2", subAccountID: "sa5", runtimeID: ""}),
			*fixInstance(instanceData{val: "A7", globalAccountID: "ga2", subAccountID: "sa6", runtimeID: "runtimeID7"}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// when
		subaccounts, err := brokerStorage.Instances().GetDistinctSubAccounts()
		require.NoError(t, err)

		// then
		assert.Equal(t, 6, len(subaccounts))
	})

	t.Run("Should fetch no distinct subaccounts from empty table of active instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// when
		subaccounts, err := brokerStorage.Instances().GetDistinctSubAccounts()
		require.NoError(t, err)

		// then
		assert.Equal(t, 0, len(subaccounts))
	})

	t.Run("Should fetch instances along with their operations", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "A1"}),
			*fixInstance(instanceData{val: "B1"}),
			*fixInstance(instanceData{val: "C1"}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		fixProvisionOps := []internal.Operation{
			fixProvisionOperation("A1"),
			fixProvisionOperation("B1"),
			fixProvisionOperation("C1"),
		}

		for _, op := range fixProvisionOps {
			err = brokerStorage.Operations().InsertOperation(op)
			require.NoError(t, err)
		}

		fixDeprovisionOps := []internal.DeprovisioningOperation{
			fixDeprovisionOperation("A1"),
			fixDeprovisionOperation("B1"),
			fixDeprovisionOperation("C1"),
		}

		for _, op := range fixDeprovisionOps {
			err = brokerStorage.Operations().InsertDeprovisioningOperation(op)
			require.NoError(t, err)
		}

		// then
		out, err := brokerStorage.Instances().FindAllJoinedWithOperations(predicate.SortAscByCreatedAt())
		require.NoError(t, err)

		require.Len(t, out, 6)

		//  checks order of instance, the oldest should be first
		sorted := sort.SliceIsSorted(out, func(i, j int) bool {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		})
		assert.True(t, sorted)

		// ignore time as this is set internally by database so will be different
		assertInstanceByIgnoreTime(t, fixInstances[0], out[0].Instance)
		assertInstanceByIgnoreTime(t, fixInstances[0], out[1].Instance)
		assertInstanceByIgnoreTime(t, fixInstances[1], out[2].Instance)
		assertInstanceByIgnoreTime(t, fixInstances[1], out[3].Instance)
		assertInstanceByIgnoreTime(t, fixInstances[2], out[4].Instance)
		assertInstanceByIgnoreTime(t, fixInstances[2], out[5].Instance)

		assertEqualOperation(t, fixProvisionOps[0], out[0])
		assertEqualOperation(t, fixDeprovisionOps[0], out[1])
		assertEqualOperation(t, fixProvisionOps[1], out[2])
		assertEqualOperation(t, fixDeprovisionOps[1], out[3])
		assertEqualOperation(t, fixProvisionOps[2], out[4])
		assertEqualOperation(t, fixDeprovisionOps[2], out[5])
	})

	t.Run("Should fetch instances based on subaccount list", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		subaccounts := []string{"sa1", "sa2", "sa3"}
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "1", subAccountID: subaccounts[0]}),
			*fixInstance(instanceData{val: "2", subAccountID: "someSU"}),
			*fixInstance(instanceData{val: "3", subAccountID: subaccounts[1]}),
			*fixInstance(instanceData{val: "4", subAccountID: subaccounts[2]}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// when
		out, err := brokerStorage.Instances().FindAllInstancesForSubAccounts(subaccounts)

		// then
		require.NoError(t, err)
		require.Len(t, out, 3)

		require.Contains(t, []string{"1", "3", "4"}, out[0].InstanceID)
		require.Contains(t, []string{"1", "3", "4"}, out[1].InstanceID)
		require.Contains(t, []string{"1", "3", "4"}, out[2].InstanceID)
	})

	t.Run("Should list instances based on page and page size", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "1"}),
			*fixInstance(instanceData{val: "2"}),
			*fixInstance(instanceData{val: "3"}),
		}
		fixOperations := []internal.Operation{
			fixture.FixProvisioningOperation("op1", "1"),
			fixture.FixProvisioningOperation("op2", "2"),
			fixture.FixProvisioningOperation("op3", "3"),
		}
		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			fixInstances[i].Reconcilable = true
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertOperation(i)
			require.NoError(t, err)
			err = brokerStorage.Instances().UpdateInstanceLastOperation(i.InstanceID, i.ID)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().List(dbmodel.InstanceFilter{PageSize: 2, Page: 1})

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 3, totalCount)

		assertInstanceByIgnoreTime(t, fixInstances[0], out[0])
		assertInstanceByIgnoreTime(t, fixInstances[1], out[1])

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{PageSize: 2, Page: 2})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 3, totalCount)

		assert.Equal(t, fixInstances[2].InstanceID, out[0].InstanceID)
	})

	t.Run("Should list instances based on filters", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "inst1"}),
			*fixInstance(instanceData{val: "inst2"}),
			*fixInstance(instanceData{val: "inst3"}),
			*fixInstance(instanceData{val: "expiredinstance", expired: true}),
		}
		fixOperations := []internal.Operation{
			fixture.FixProvisioningOperation("op1", "inst1"),
			fixture.FixProvisioningOperation("op2", "inst2"),
			fixture.FixProvisioningOperation("op3", "inst3"),
			fixture.FixProvisioningOperation("op4", "expiredinstance"),
		}
		fixBinding := fixture.FixBinding("binding1")
		fixBinding.InstanceID = fixInstances[0].InstanceID
		err = brokerStorage.Bindings().Insert(&fixBinding)
		require.NoError(t, err)
		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertOperation(i)
			require.NoError(t, err)
			err = brokerStorage.Instances().UpdateInstanceLastOperation(i.InstanceID, i.ID)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().List(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[0].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{GlobalAccountIDs: []string{fixInstances[1].GlobalAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[1].SubAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{RuntimeIDs: []string{fixInstances[1].RuntimeID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Plans: []string{fixInstances[1].ServicePlanName}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Shoots: []string{"Shoot-inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Regions: []string{"inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(true)})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[3].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(false)})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, 3, totalCount)

		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{BindingExists: ptr.Bool(true)})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

	})

	t.Run("Should list instances with proper subaccount state info", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "inst1", subAccountID: "common-subaccount"}),
			*fixInstance(instanceData{val: "inst2"}),
			*fixInstance(instanceData{val: "inst3"}),
			*fixInstance(instanceData{val: "expiredinstance", expired: true}),
			*fixInstance(instanceData{val: "inst4", subAccountID: "common-subaccount"}),
		}
		fixOperations := []internal.Operation{
			fixture.FixProvisioningOperation("op1", "inst1"),
			fixture.FixProvisioningOperation("op2", "inst2"),
			fixture.FixProvisioningOperation("op3", "inst3"),
			fixture.FixProvisioningOperation("op4", "expiredinstance"),
			fixture.FixProvisioningOperation("op5", "inst4"),
		}
		// there is no record for subaccount used by inst3 by purpose
		fixSubaccountStates := []internal.SubaccountState{
			{
				ID:                fixInstances[0].SubAccountID,
				BetaEnabled:       "true",
				UsedForProduction: "NOT_SET",
				ModifiedAt:        10,
			},
			{
				ID:                fixInstances[1].SubAccountID,
				BetaEnabled:       "true",
				UsedForProduction: "USED_FOR_PRODUCTION",
				ModifiedAt:        20,
			},
			{
				ID:                fixInstances[3].SubAccountID,
				BetaEnabled:       "true",
				UsedForProduction: "",
				ModifiedAt:        30,
			},
			{
				ID:                "not-existing-subaccount",
				BetaEnabled:       "true",
				UsedForProduction: "USED_FOR_PRODUCTION",
				ModifiedAt:        40,
			},
		}
		for _, s := range fixSubaccountStates {
			err = brokerStorage.SubaccountStates().UpsertState(s)
			require.NoError(t, err)
		}

		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertOperation(i)
			require.NoError(t, err)
			err = brokerStorage.Instances().UpdateInstanceLastOperation(i.InstanceID, i.ID)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[0].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixSubaccountStates[0].BetaEnabled, out[0].BetaEnabled)
		assert.Equal(t, fixSubaccountStates[0].UsedForProduction, out[0].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[1].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixSubaccountStates[1].BetaEnabled, out[0].BetaEnabled)
		assert.Equal(t, fixSubaccountStates[1].UsedForProduction, out[0].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[2].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[2].InstanceID, out[0].InstanceID)
		assert.Empty(t, out[0].BetaEnabled)
		assert.Empty(t, out[0].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[3].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[3].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixSubaccountStates[3].BetaEnabled, out[0].BetaEnabled)
		assert.Empty(t, out[0].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[0].SubAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 2, totalCount)

		assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixSubaccountStates[0].BetaEnabled, out[0].BetaEnabled)
		assert.Equal(t, fixSubaccountStates[0].UsedForProduction, out[0].UsedForProduction)
		assert.Equal(t, fixSubaccountStates[0].BetaEnabled, out[1].BetaEnabled)
		assert.Equal(t, fixSubaccountStates[0].UsedForProduction, out[1].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{GlobalAccountIDs: []string{fixInstances[1].GlobalAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixSubaccountStates[1].BetaEnabled, out[0].BetaEnabled)
		assert.Equal(t, fixSubaccountStates[1].UsedForProduction, out[0].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[1].SubAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)
		assert.Equal(t, fixSubaccountStates[1].BetaEnabled, out[0].BetaEnabled)
		assert.Equal(t, fixSubaccountStates[1].UsedForProduction, out[0].UsedForProduction)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{Expired: ptr.Bool(true)})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[3].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().ListWithSubaccountState(dbmodel.InstanceFilter{Expired: ptr.Bool(false)})
		require.NoError(t, err)
		require.Equal(t, 4, count)
		require.Equal(t, 4, totalCount)

	})

	t.Run("Should list instances based on filters", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "inst1"}),
			*fixInstance(instanceData{val: "inst2"}),
			*fixInstance(instanceData{val: "inst3"}),
			*fixInstance(instanceData{val: "expiredinstance", expired: true}),
		}
		fixOperations := []internal.Operation{
			fixture.FixProvisioningOperation("op1", "inst1"),
			fixture.FixProvisioningOperation("op2", "inst2"),
			fixture.FixProvisioningOperation("op3", "inst3"),
			fixture.FixProvisioningOperation("op4", "expiredinstance"),
		}
		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertOperation(i)
			require.NoError(t, err)
			err = brokerStorage.Instances().UpdateInstanceLastOperation(i.InstanceID, i.ID)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().List(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[0].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{GlobalAccountIDs: []string{fixInstances[1].GlobalAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[1].SubAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{RuntimeIDs: []string{fixInstances[1].RuntimeID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Plans: []string{fixInstances[1].ServicePlanName}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Shoots: []string{"Shoot-inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Regions: []string{"inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(true)})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[3].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(false)})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, 3, totalCount)

	})

	t.Run("Should list trial instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		inst1 := fixInstance(instanceData{val: "inst1"})
		inst2 := fixInstance(instanceData{val: "inst2", trial: true, expired: true})
		inst3 := fixInstance(instanceData{val: "inst3", trial: true})
		inst4 := fixInstance(instanceData{val: "inst4"})
		fixInstances := []internal.Instance{*inst1, *inst2, *inst3, *inst4}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// inst1 is in succeeded state
		provOp1 := fixProvisionOperation("inst1")
		provOp1.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp1)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst1", provOp1.ID)
		require.NoError(t, err)

		// inst2 is in succeeded state
		provOp2 := fixProvisionOperation("inst2")
		provOp2.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp2)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst2", provOp2.ID)
		require.NoError(t, err)

		// inst3 is in succeeded state
		provOp3 := fixProvisionOperation("inst3")
		provOp3.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp3)
		require.NoError(t, err)
		deprovOp3 := fixDeprovisionOperation("inst3")
		deprovOp3.Temporary = true
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = brokerStorage.Operations().InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst3", deprovOp3.ID)
		require.NoError(t, err)

		// inst4 is in failed state
		provOp4 := fixProvisionOperation("inst4")
		provOp4.State = domain.Failed
		err = brokerStorage.Operations().InsertOperation(provOp4)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst4", provOp4.ID)
		require.NoError(t, err)

		// when
		nonExpiredTrialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{broker.TrialPlanID}, Expired: &[]bool{false}[0]}
		out, count, totalCount, err := brokerStorage.Instances().List(nonExpiredTrialInstancesFilter)

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)
		require.Equal(t, inst3.InstanceID, out[0].InstanceID)

		// when
		trialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{broker.TrialPlanID}}
		out, count, totalCount, err = brokerStorage.Instances().List(trialInstancesFilter)

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 2, totalCount)
		require.Equal(t, inst2.InstanceID, out[0].InstanceID)
		require.Equal(t, inst3.InstanceID, out[1].InstanceID)
	})

	t.Run("Should list regular instances and not completely deprovisioned instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		inst1 := fixInstance(instanceData{val: "inst1", deletedAt: time.Now()})
		inst2 := fixInstance(instanceData{val: "inst2", trial: true, expired: true, deletedAt: time.Now()})
		inst3 := fixInstance(instanceData{val: "inst3", trial: true, deletedAt: time.Time{}})
		inst4 := fixInstance(instanceData{val: "inst4", deletedAt: time.Now()})
		fixInstances := []internal.Instance{*inst1, *inst2, *inst3, *inst4}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// inst1 is in succeeded state
		provOp1 := fixProvisionOperation("inst1")
		provOp1.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp1)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst1", provOp1.ID)
		require.NoError(t, err)

		// inst2 is in succeeded state
		provOp2 := fixProvisionOperation("inst2")
		provOp2.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp2)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst2", provOp2.ID)
		require.NoError(t, err)

		// inst3 is in succeeded state
		provOp3 := fixProvisionOperation("inst3")
		provOp3.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp3)
		require.NoError(t, err)
		deprovOp3 := fixDeprovisionOperation("inst3")
		deprovOp3.Temporary = true
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = brokerStorage.Operations().InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst3", provOp3.ID)
		require.NoError(t, err)

		// inst4 is in failed state
		provOp4 := fixProvisionOperation("inst4")
		provOp4.State = domain.Failed
		err = brokerStorage.Operations().InsertOperation(provOp4)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst4", provOp4.ID)
		require.NoError(t, err)

		// when
		emptyFilter := dbmodel.InstanceFilter{}
		out, count, _, err := brokerStorage.Instances().List(emptyFilter)
		var notCompletelyDeleted int
		for _, instance := range out {
			if !instance.DeletedAt.IsZero() {
				notCompletelyDeleted += 1
			}
		}

		// then
		require.NoError(t, err)
		require.Equal(t, 4, count)
		require.Equal(t, 3, notCompletelyDeleted)
	})

	t.Run("Should list not completely deprovisioned instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		inst1 := fixInstance(instanceData{val: "inst1", deletedAt: time.Now()})
		inst2 := fixInstance(instanceData{val: "inst2", trial: true, expired: true, deletedAt: time.Now()})
		inst3 := fixInstance(instanceData{val: "inst3", trial: true, deletedAt: time.Time{}})
		inst4 := fixInstance(instanceData{val: "inst4", deletedAt: time.Now()})
		fixInstances := []internal.Instance{*inst1, *inst2, *inst3, *inst4}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// inst1 is in succeeded state
		provOp1 := fixProvisionOperation("inst1")
		provOp1.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp1)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst1", provOp1.ID)
		require.NoError(t, err)

		// inst2 is in succeeded state
		provOp2 := fixProvisionOperation("inst2")
		provOp2.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp2)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst2", provOp2.ID)
		require.NoError(t, err)

		// inst3 is in succeeded state
		provOp3 := fixProvisionOperation("inst3")
		provOp3.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp3)
		require.NoError(t, err)
		deprovOp3 := fixDeprovisionOperation("inst3")
		deprovOp3.Temporary = true
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = brokerStorage.Operations().InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst3", provOp3.ID)
		require.NoError(t, err)

		// inst4 is in failed state
		provOp4 := fixProvisionOperation("inst4")
		provOp4.State = domain.Failed
		err = brokerStorage.Operations().InsertOperation(provOp4)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst4", provOp4.ID)
		require.NoError(t, err)

		// when
		notCompletelyDeletedFilter := dbmodel.InstanceFilter{DeletionAttempted: &[]bool{true}[0]}

		_, notCompletelyDeleted, _, err := brokerStorage.Instances().List(notCompletelyDeletedFilter)

		// then
		require.NoError(t, err)
		require.Equal(t, 3, notCompletelyDeleted)
	})

	t.Run("Should list suspended instances", func(t *testing.T) {
		storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// populate database with samples
		inst1 := fixInstance(instanceData{val: "inst1"})
		inst2 := fixInstance(instanceData{val: "inst2"})
		inst3 := fixInstance(instanceData{val: "inst3"})
		inst3.Parameters.ErsContext.Active = ptr.Bool(false)
		inst4 := fixInstance(instanceData{val: "inst4"})
		fixInstances := []internal.Instance{*inst1, *inst2, *inst3, *inst4}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// inst1 is in succeeded state
		provOp1 := fixProvisionOperation("inst1")
		provOp1.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp1)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst1", provOp1.ID)
		require.NoError(t, err)

		// inst2 is in succeeded state
		provOp2 := fixProvisionOperation("inst2")
		provOp2.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp2)
		require.NoError(t, err)
		deprovOp2 := fixDeprovisionOperation("inst2")
		deprovOp2.Temporary = true
		deprovOp2.State = domain.Succeeded
		deprovOp2.CreatedAt = deprovOp2.CreatedAt.Add(2 * time.Minute)
		err = brokerStorage.Operations().InsertDeprovisioningOperation(deprovOp2)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst2", provOp2.ID)
		require.NoError(t, err)

		// inst3 is in succeeded state
		provOp3 := fixProvisionOperation("inst3")
		provOp3.State = domain.Succeeded
		err = brokerStorage.Operations().InsertOperation(provOp3)
		require.NoError(t, err)
		deprovOp3 := fixDeprovisionOperation("inst3")
		deprovOp3.Temporary = true
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = brokerStorage.Operations().InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst3", provOp3.ID)
		require.NoError(t, err)

		// inst4 is in failed state
		provOp4 := fixProvisionOperation("inst4")
		provOp4.State = domain.Failed
		err = brokerStorage.Operations().InsertOperation(provOp4)
		require.NoError(t, err)
		err = brokerStorage.Instances().UpdateInstanceLastOperation("inst4", provOp4.ID)
		require.NoError(t, err)

		// when
		suspendedFilter := dbmodel.InstanceFilter{Suspended: ptr.Bool(true)}

		_, suspended, _, err := brokerStorage.Instances().List(suspendedFilter)

		// then
		require.NoError(t, err)
		require.Equal(t, 1, suspended)
	})
}

func TestInstanceStorage_ListInstancesUsingLastOperationID(t *testing.T) {
	// given
	cfg := brokerStorageDatabaseTestConfig()
	storageCleanup, brokerStorage, err := storage.GetStorageForTest(cfg)
	require.NoError(t, err)
	require.NotNil(t, brokerStorage)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()
	instanceStorage := brokerStorage.Instances()
	operationStorage := brokerStorage.Operations()

	// first instance, one provisioning, one update
	instance0 := fixInstance(instanceData{val: "inst1"})
	_ = instanceStorage.Insert(*instance0)
	operation0_0 := fixProvisionOperation(instance0.InstanceID)
	operation0_1 := fixture.FixUpdatingOperation("op0_1", instance0.InstanceID).Operation
	operation0_1.State = domain.InProgress

	_ = operationStorage.InsertOperation(operation0_0)
	_ = operationStorage.InsertOperation(operation0_1)
	_ = instanceStorage.UpdateInstanceLastOperation(instance0.InstanceID, operation0_1.ID)

	// second instance, one provisioning, 1 update
	instance1 := fixInstance(instanceData{val: fmt.Sprintf("inst2")})
	_ = instanceStorage.Insert(*instance1)

	operation1_0 := fixProvisionOperation(instance1.InstanceID)

	_ = operationStorage.InsertOperation(operation1_0)

	op := fixture.FixUpdatingOperation(fmt.Sprintf("op1__%s", instance1.InstanceID), instance1.InstanceID).Operation
	op.CreatedAt = time.Now().Add(-3 * time.Second)
	op.State = domain.Succeeded
	_ = operationStorage.InsertOperation(op)

	operation1_1 := fixture.FixUpdatingOperation("op1_1", instance1.InstanceID).Operation
	operation1_1.State = domain.InProgress
	_ = operationStorage.InsertOperation(operation1_1)
	_ = instanceStorage.UpdateInstanceLastOperation(instance1.InstanceID, operation1_1.ID)

	// third instance, one provisioning, one deprovisioning
	instance2 := fixInstance(instanceData{val: "inst3"})
	_ = instanceStorage.Insert(*instance2)

	operation2_0 := fixProvisionOperation(instance2.InstanceID)
	operation2_1 := fixDeprovisionOperation(instance2.InstanceID).Operation
	operation2_1.State = domain.InProgress

	_ = operationStorage.InsertOperation(operation2_0)
	_ = operationStorage.InsertOperation(operation2_1)
	_ = instanceStorage.UpdateInstanceLastOperation(instance2.InstanceID, operation2_1.ID)

	// when
	got, _, _, err := instanceStorage.ListWithSubaccountState(dbmodel.InstanceFilter{
		States:   []dbmodel.InstanceState{dbmodel.InstanceUpdating},
		PageSize: 10,
		Page:     1,
	})
	assert.Equal(t, 2, len(got))

	got, _, _, err = instanceStorage.ListWithSubaccountState(dbmodel.InstanceFilter{
		States:   []dbmodel.InstanceState{dbmodel.InstanceUpdating},
		PageSize: 1,
		Page:     1,
	})
	assert.Equal(t, 1, len(got))

	// when
	got, _, _, err = instanceStorage.ListWithSubaccountState(dbmodel.InstanceFilter{
		States:   []dbmodel.InstanceState{dbmodel.InstanceDeprovisioning},
		PageSize: 10,
		Page:     1,
	})
	assert.Equal(t, 1, len(got))
	assert.Equal(t, instance2.InstanceID, got[0].InstanceID)

}

func TestInstance_ModeGCM(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################")

	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// when
	instanceID := "instance-gcm"
	instanceGCM := fixInstance(instanceData{val: instanceID})

	err = brokerStorage.Instances().Insert(*instanceGCM)
	require.NoError(t, err)

	retrievedInstance, err := brokerStorage.Instances().GetByID(instanceID)

	// then
	require.NoError(t, err)
	assert.Equal(t, instanceGCM.Parameters.ErsContext.SMOperatorCredentials.ClientSecret, retrievedInstance.Parameters.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, instanceGCM.Parameters.ErsContext.SMOperatorCredentials.ClientID, retrievedInstance.Parameters.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, instanceGCM.Parameters.Parameters.Kubeconfig, retrievedInstance.Parameters.Parameters.Kubeconfig)
}

func assertInstanceByIgnoreTime(t *testing.T, want, got internal.Instance) {
	t.Helper()
	want.CreatedAt, got.CreatedAt = time.Time{}, time.Time{}
	want.UpdatedAt, got.UpdatedAt = time.Time{}, time.Time{}
	want.DeletedAt, got.DeletedAt = time.Time{}, time.Time{}
	want.ExpiredAt, got.ExpiredAt = nil, nil

	assert.EqualValues(t, want, got)
}

func assertEqualOperation(t *testing.T, want interface{}, got internal.InstanceWithOperation) {
	t.Helper()
	switch want := want.(type) {
	case internal.ProvisioningOperation:
		assert.EqualValues(t, internal.OperationTypeProvision, got.Type.String)
		assert.EqualValues(t, want.State, got.State.String)
		assert.EqualValues(t, want.Description, got.Description.String)
	case internal.DeprovisioningOperation:
		assert.EqualValues(t, internal.OperationTypeDeprovision, got.Type.String)
		assert.EqualValues(t, want.State, got.State.String)
		assert.EqualValues(t, want.Description, got.Description.String)
	}
}

type instanceData struct {
	val             string
	globalAccountID string
	subAccountID    string
	runtimeID       string
	expired         bool
	trial           bool
	deletedAt       time.Time
}

func fixInstance(testData instanceData) *internal.Instance {
	var (
		gaid string
		suid string
	)

	if testData.globalAccountID != "" {
		gaid = testData.globalAccountID
	} else {
		gaid = testData.val
	}

	if testData.subAccountID != "" {
		suid = testData.subAccountID
	} else {
		suid = testData.val
	}

	instance := fixture.FixInstance(testData.val)
	instance.GlobalAccountID = gaid
	instance.SubscriptionGlobalAccountID = gaid
	instance.SubAccountID = suid
	if testData.trial {
		instance.ServicePlanID = broker.TrialPlanID
		instance.ServicePlanName = broker.TrialPlanName
	} else {
		instance.ServiceID = testData.val
		instance.ServiceName = testData.val
	}
	instance.ServicePlanName = testData.val
	instance.DashboardURL = fmt.Sprintf("https://console.%s.kyma.local", testData.val)
	instance.ProviderRegion = testData.val
	instance.Parameters.ErsContext.SubAccountID = suid
	instance.Parameters.ErsContext.GlobalAccountID = gaid
	serviceManagerOperatorCredentials := internal.ServiceManagerOperatorCredentials{
		ClientID:     fmt.Sprintf("client-id-%s", testData.val),
		ClientSecret: fmt.Sprintf("client-secret-%s", testData.val),
	}
	instance.Parameters.ErsContext.SMOperatorCredentials = &serviceManagerOperatorCredentials
	instance.Parameters.Parameters.Kubeconfig = fmt.Sprintf("kubeconfig-%s", testData.val)

	instance.InstanceDetails = internal.InstanceDetails{}
	if testData.expired {
		instance.ExpiredAt = ptr.Time(time.Now().Add(-10 * time.Hour))
	}
	if !testData.deletedAt.IsZero() {
		instance.DeletedAt = testData.deletedAt
	}
	return &instance
}

func fixProvisionOperation(instanceId string) internal.Operation {
	operationId := fmt.Sprintf("%s-%d", instanceId, rand.Int())
	return fixture.FixProvisioningOperation(operationId, instanceId)

}
func fixDeprovisionOperation(instanceId string) internal.DeprovisioningOperation {
	operationId := fmt.Sprintf("%s-%d", instanceId, rand.Int())
	return fixture.FixDeprovisioningOperation(operationId, instanceId)
}
