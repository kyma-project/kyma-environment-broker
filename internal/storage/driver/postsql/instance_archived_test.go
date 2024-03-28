package postsql_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/pivotal-cf/brokerapi/v8/domain"

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
