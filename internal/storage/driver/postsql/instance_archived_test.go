package postsql_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/pivotal-cf/brokerapi/v12/domain"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceArchived(t *testing.T) {

	t.Run("insert and get instance archived", func(t *testing.T) {
		// given
		time1, _ := time.Parse("2006-01-02T15:04:05", "2024-01-05T9:53:44")
		givenInstance1 := internal.InstanceArchived{
			InstanceID:                    "instance-id1",
			GlobalAccountID:               "gaic",
			SubaccountID:                  "said",
			SubscriptionGlobalAccountID:   "gaic",
			PlanID:                        "plan-id00",
			PlanName:                      "plan-name00",
			SubaccountRegion:              "cf-eu10",
			Region:                        "westeurope",
			Provider:                      "azure",
			LastRuntimeID:                 "runtime-id",
			InternalUser:                  true,
			ShootName:                     "shoot-00001",
			ProvisioningStartedAt:         time1,
			ProvisioningFinishedAt:        time1.Add(10 * time.Minute),
			ProvisioningState:             domain.Succeeded,
			FirstDeprovisioningStartedAt:  time1.Add(time.Hour),
			FirstDeprovisioningFinishedAt: time1.Add(3 * time.Hour),
			LastDeprovisioningFinishedAt:  time1.Add(4 * time.Hour),
		}
		time2, _ := time.Parse("2006-01-02T15:04:05", "2022-12-05T18:21:41")
		givenInstance2 := internal.InstanceArchived{
			InstanceID:                    "instance-id2",
			GlobalAccountID:               "gaic1",
			SubaccountID:                  "said1",
			SubscriptionGlobalAccountID:   "gaic1",
			PlanID:                        "plan-id001",
			PlanName:                      "plan-name001",
			SubaccountRegion:              "cf-eu20",
			Region:                        "westeurope",
			Provider:                      "azure",
			LastRuntimeID:                 "runtime-id1",
			InternalUser:                  false,
			ShootName:                     "shoot-00002",
			ProvisioningStartedAt:         time2.Add(-1 * time.Hour),
			ProvisioningFinishedAt:        time2.Add(10 * time.Minute),
			ProvisioningState:             domain.Succeeded,
			FirstDeprovisioningStartedAt:  time2.Add(time.Hour),
			FirstDeprovisioningFinishedAt: time2.Add(3 * time.Hour),
			LastDeprovisioningFinishedAt:  time2.Add(4 * time.Hour),
		}
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
		db := brokerStorage.InstancesArchived()

		// when
		err = db.Insert(givenInstance1)
		require.NoError(t, err)
		err = db.Insert(givenInstance2)
		require.NoError(t, err)

		got1, err := db.GetByInstanceID("instance-id1")
		require.NoError(t, err)

		got2, err := db.GetByInstanceID("instance-id2")
		require.NoError(t, err)

		// then
		assertInstanceArchived(t, givenInstance1, got1)
		assertInstanceArchived(t, givenInstance2, got2)
	})

	t.Run("get total number of instances archived for global accounts", func(t *testing.T) {
		// given
		givenInstance1 := fixInstanceArchive(instanceArchiveData{
			InstanceID:        "instance-id1",
			GlobalAccountID:   "gaidA",
			PlanID:            broker.FreemiumPlanID,
			PlanName:          broker.FreemiumPlanName,
			ProvisioningState: domain.Succeeded,
		})
		givenInstance2 := fixInstanceArchive(instanceArchiveData{
			InstanceID:        "instance-id2",
			GlobalAccountID:   "gaidA",
			PlanID:            broker.FreemiumPlanID,
			PlanName:          broker.FreemiumPlanName,
			ProvisioningState: domain.Succeeded,
		})
		givenInstance3 := fixInstanceArchive(instanceArchiveData{
			InstanceID:        "instance-id3",
			GlobalAccountID:   "gaidA",
			PlanID:            broker.FreemiumPlanID,
			PlanName:          broker.FreemiumPlanName,
			ProvisioningState: domain.Failed,
		})
		givenInstance4 := fixInstanceArchive(instanceArchiveData{
			InstanceID:        "instance-id4",
			GlobalAccountID:   "gaidB",
			PlanID:            broker.FreemiumPlanID,
			PlanName:          broker.FreemiumPlanName,
			ProvisioningState: domain.Succeeded,
		})
		givenInstance5 := fixInstanceArchive(instanceArchiveData{
			InstanceID:        "instance-id5",
			GlobalAccountID:   "gaidB",
			PlanID:            broker.TrialPlanID,
			PlanName:          broker.TrialPlanName,
			ProvisioningState: domain.Succeeded,
		})
		givenInstance6 := fixInstanceArchive(instanceArchiveData{
			InstanceID:        "instance-id6",
			GlobalAccountID:   "gaidB",
			PlanID:            broker.TrialPlanID,
			PlanName:          broker.TrialPlanName,
			ProvisioningState: domain.Failed,
		})

		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
		db := brokerStorage.InstancesArchived()

		// when
		err = db.Insert(givenInstance1)
		require.NoError(t, err)
		err = db.Insert(givenInstance2)
		require.NoError(t, err)
		err = db.Insert(givenInstance3)
		require.NoError(t, err)
		err = db.Insert(givenInstance4)
		require.NoError(t, err)
		err = db.Insert(givenInstance5)
		require.NoError(t, err)
		err = db.Insert(givenInstance6)
		require.NoError(t, err)

		// then
		require.NoError(t, err)
		numberOfInstancesA, err := db.TotalNumberOfInstancesArchivedForGlobalAccountID("gaidA", broker.FreemiumPlanID)
		require.NoError(t, err)
		numberOfInstancesB, err := db.TotalNumberOfInstancesArchivedForGlobalAccountID("gaidB", broker.FreemiumPlanID)
		require.NoError(t, err)
		numberOfInstancesC, err := db.TotalNumberOfInstancesArchivedForGlobalAccountID("gaidC", broker.FreemiumPlanID)
		require.NoError(t, err)

		assert.Equal(t, 2, numberOfInstancesA)
		assert.Equal(t, 1, numberOfInstancesB)
		assert.Equal(t, 0, numberOfInstancesC)
	})

	t.Run("Should list instances based on page and page size", func(t *testing.T) {
		// given
		givenInstance1 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id1",
			GlobalAccountID: "gaidA",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
		})
		givenInstance2 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id2",
			GlobalAccountID: "gaidA",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
		})
		givenInstance3 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id3",
			GlobalAccountID: "gaidA",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
		})

		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
		db := brokerStorage.InstancesArchived()

		err = db.Insert(givenInstance1)
		require.NoError(t, err)
		err = db.Insert(givenInstance2)
		require.NoError(t, err)
		err = db.Insert(givenInstance3)
		require.NoError(t, err)

		// when
		out, count, totalCount, err := db.List(dbmodel.InstanceFilter{PageSize: 2, Page: 1})

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 3, totalCount)

		assertInstanceArchived(t, givenInstance1, out[0])
		assertInstanceArchived(t, givenInstance2, out[1])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{PageSize: 2, Page: 2})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 3, totalCount)

		assertInstanceArchived(t, givenInstance3, out[0])
	})

	t.Run("Should list instances based on filters", func(t *testing.T) {
		// given
		givenInstance1 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id1",
			GlobalAccountID: "gaidA",
			SubaccountID:    "saidA1",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
			Region:          "westeurope",
			LastRuntimeID:   "runtime-id1",
			ShootName:       "shoot-1",
		})
		givenInstance2 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id2",
			GlobalAccountID: "gaidA",
			SubaccountID:    "saidA1",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
			Region:          "westeurope",
			LastRuntimeID:   "runtime-id2",
			ShootName:       "shoot-2",
		})
		givenInstance3 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id3",
			GlobalAccountID: "gaidA",
			SubaccountID:    "saidA2",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
			Region:          "westeurope",
			LastRuntimeID:   "runtime-id3",
			ShootName:       "shoot-3",
		})
		givenInstance4 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id4",
			GlobalAccountID: "gaidB",
			SubaccountID:    "saidB1",
			PlanID:          broker.FreemiumPlanID,
			PlanName:        broker.FreemiumPlanName,
			Region:          "westeurope",
			LastRuntimeID:   "runtime-id4",
			ShootName:       "shoot-4",
		})
		givenInstance5 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id5",
			GlobalAccountID: "gaidB",
			SubaccountID:    "saidB2",
			PlanID:          broker.TrialPlanID,
			PlanName:        broker.TrialPlanName,
			Region:          "easteurope",
			LastRuntimeID:   "runtime-id5",
			ShootName:       "shoot-5",
		})
		givenInstance6 := fixInstanceArchive(instanceArchiveData{
			InstanceID:      "instance-id6",
			GlobalAccountID: "gaidB",
			SubaccountID:    "saidB2",
			PlanID:          broker.TrialPlanID,
			PlanName:        broker.TrialPlanName,
			Region:          "easteurope",
			LastRuntimeID:   "runtime-id6",
			ShootName:       "shoot-6",
		})

		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
		db := brokerStorage.InstancesArchived()

		err = db.Insert(givenInstance1)
		require.NoError(t, err)
		err = db.Insert(givenInstance2)
		require.NoError(t, err)
		err = db.Insert(givenInstance3)
		require.NoError(t, err)
		err = db.Insert(givenInstance4)
		require.NoError(t, err)
		err = db.Insert(givenInstance5)
		require.NoError(t, err)
		err = db.Insert(givenInstance6)
		require.NoError(t, err)

		// when
		out, count, totalCount, err := db.List(dbmodel.InstanceFilter{InstanceIDs: []string{"instance-id4"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assertInstanceArchived(t, givenInstance4, out[0])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{GlobalAccountIDs: []string{"gaidB"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, 3, totalCount)

		assertInstanceArchived(t, givenInstance4, out[0])
		assertInstanceArchived(t, givenInstance5, out[1])
		assertInstanceArchived(t, givenInstance6, out[2])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{SubAccountIDs: []string{"saidB1"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assertInstanceArchived(t, givenInstance4, out[0])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{Plans: []string{broker.TrialPlanName}})

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 2, totalCount)

		assertInstanceArchived(t, givenInstance5, out[0])
		assertInstanceArchived(t, givenInstance6, out[1])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{Regions: []string{"easteurope"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 2, totalCount)

		assertInstanceArchived(t, givenInstance5, out[0])
		assertInstanceArchived(t, givenInstance6, out[1])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{RuntimeIDs: []string{"runtime-id3"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assertInstanceArchived(t, givenInstance3, out[0])

		// when
		out, count, totalCount, err = db.List(dbmodel.InstanceFilter{Shoots: []string{"shoot-5"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assertInstanceArchived(t, givenInstance5, out[0])
	})
}

func assertInstanceArchived(t *testing.T, expected internal.InstanceArchived, got internal.InstanceArchived) {
	t.Helper()

	assert.True(t, expected.FirstDeprovisioningStartedAt.Equal(got.FirstDeprovisioningStartedAt))
	assert.True(t, expected.FirstDeprovisioningFinishedAt.Equal(got.FirstDeprovisioningFinishedAt))
	assert.True(t, expected.LastDeprovisioningFinishedAt.Equal(got.LastDeprovisioningFinishedAt))
	assert.True(t, expected.ProvisioningStartedAt.Equal(got.ProvisioningStartedAt), fmt.Sprintf("%v %v", expected.ProvisioningStartedAt, got.ProvisioningStartedAt))
	assert.True(t, expected.ProvisioningFinishedAt.Equal(got.ProvisioningFinishedAt), fmt.Sprintf("%v %v", expected.ProvisioningFinishedAt, got.ProvisioningFinishedAt))
	assert.Equal(t, expected.ProvisioningState, got.ProvisioningState)

	expected.ProvisioningFinishedAt = got.ProvisioningFinishedAt
	expected.ProvisioningStartedAt = got.ProvisioningStartedAt
	expected.ProvisioningState = got.ProvisioningState
	expected.LastDeprovisioningFinishedAt = got.LastDeprovisioningFinishedAt
	expected.FirstDeprovisioningFinishedAt = got.FirstDeprovisioningFinishedAt
	expected.FirstDeprovisioningStartedAt = got.FirstDeprovisioningStartedAt

	assert.Equal(t, expected, got)
}

type instanceArchiveData struct {
	InstanceID        string
	GlobalAccountID   string
	SubaccountID      string
	PlanID            string
	PlanName          string
	Region            string
	LastRuntimeID     string
	ShootName         string
	ProvisioningState domain.LastOperationState
}

func fixInstanceArchive(testData instanceArchiveData) internal.InstanceArchived {
	provisioningTime, _ := time.Parse("2006-01-02T15:04:05", "2022-12-05T18:22:41")
	if testData.ProvisioningState == "" {
		testData.ProvisioningState = domain.Succeeded
	}
	if testData.SubaccountID == "" {
		testData.SubaccountID = testData.GlobalAccountID
	}
	if testData.Region == "" {
		testData.Region = "westeurope"
	}
	if testData.LastRuntimeID == "" {
		testData.LastRuntimeID = "runtime-id"
	}
	if testData.ShootName == "" {
		testData.ShootName = "shoot-0000"
	}
	return internal.InstanceArchived{
		InstanceID:                    testData.InstanceID,
		GlobalAccountID:               testData.GlobalAccountID,
		SubaccountID:                  testData.SubaccountID,
		SubscriptionGlobalAccountID:   testData.GlobalAccountID,
		PlanID:                        testData.PlanID,
		PlanName:                      testData.PlanName,
		SubaccountRegion:              "cf-eu20",
		Region:                        testData.Region,
		Provider:                      "azure",
		LastRuntimeID:                 testData.LastRuntimeID,
		InternalUser:                  false,
		ShootName:                     testData.ShootName,
		ProvisioningStartedAt:         provisioningTime.Add(-1 * time.Hour),
		ProvisioningFinishedAt:        provisioningTime.Add(10 * time.Minute),
		ProvisioningState:             testData.ProvisioningState,
		FirstDeprovisioningStartedAt:  provisioningTime.Add(time.Hour),
		FirstDeprovisioningFinishedAt: provisioningTime.Add(3 * time.Hour),
		LastDeprovisioningFinishedAt:  provisioningTime.Add(4 * time.Hour),
	}
}
