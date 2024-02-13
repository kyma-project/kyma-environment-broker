package postsql_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestration(t *testing.T) {

	t.Run("Orchestrations", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		givenOrchestration := fixture.FixOrchestration("test")
		givenOrchestration.Type = orchestration.UpgradeKymaOrchestration
		givenOrchestration.State = "test"
		givenOrchestration.Description = "test"
		givenOrchestration.Parameters.DryRun = true

		svc := brokerStorage.Orchestrations()

		err = svc.Insert(givenOrchestration)
		require.NoError(t, err)

		// when
		gotOrchestration, err := svc.GetByID("test")
		require.NoError(t, err)
		assert.Equal(t, givenOrchestration.Parameters, gotOrchestration.Parameters)
		assert.Equal(t, orchestration.UpgradeKymaOrchestration, gotOrchestration.Type)

		gotOrchestration.Description = "new modified description 1"
		err = svc.Update(givenOrchestration)
		require.NoError(t, err)

		err = svc.Insert(givenOrchestration)
		assertError(t, dberr.CodeAlreadyExists, err)

		l, count, totalCount, err := svc.List(dbmodel.OrchestrationFilter{PageSize: 10, Page: 1})
		require.NoError(t, err)
		assert.Len(t, l, 1)
		assert.Equal(t, 1, count)
		assert.Equal(t, 1, totalCount)

		l, c, tc, err := svc.List(dbmodel.OrchestrationFilter{States: []string{"test"}, Types: []string{string(orchestration.UpgradeKymaOrchestration)}})
		require.NoError(t, err)
		assert.Len(t, l, 1)
		assert.Equal(t, 1, c)
		assert.Equal(t, 1, tc)
	})
}
