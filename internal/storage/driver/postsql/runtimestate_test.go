package postsql_test

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeState(t *testing.T) {

	t.Run("should insert and fetch RuntimeState", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		fixID := "test"
		givenRuntimeState := fixture.FixRuntimeState(fixID, fixID, fixID)
		givenRuntimeState.KymaConfig.Version = fixID
		givenRuntimeState.ClusterConfig.KubernetesVersion = fixID

		svc := brokerStorage.RuntimeStates()

		err = svc.Insert(givenRuntimeState)
		require.NoError(t, err)

		runtimeStates, err := svc.ListByRuntimeID(fixID)
		require.NoError(t, err)
		assert.Len(t, runtimeStates, 1)
		assert.Equal(t, fixID, runtimeStates[0].KymaConfig.Version)
		assert.Equal(t, fixID, runtimeStates[0].ClusterConfig.KubernetesVersion)

		state, err := svc.GetByOperationID(fixID)
		require.NoError(t, err)
		assert.Equal(t, fixID, state.KymaConfig.Version)
		assert.Equal(t, fixID, state.ClusterConfig.KubernetesVersion)
	})

	t.Run("should fetch latest RuntimeState with OIDC config", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		fixRuntimeID := "runtimeID"
		expectedOIDCConfig := gqlschema.OIDCConfigInput{
			ClientID:       "clientID",
			GroupsClaim:    "groups",
			IssuerURL:      "https://issuer.url",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "sub",
			UsernamePrefix: "-",
		}

		fixRuntimeStateID1 := "runtimestate1"
		fixOperationID1 := "operation1"
		runtimeStateWithOIDCConfig := fixture.FixRuntimeState(fixRuntimeStateID1, fixRuntimeID, fixOperationID1)
		runtimeStateWithOIDCConfig.ClusterConfig.OidcConfig = &gqlschema.OIDCConfigInput{
			ClientID:       expectedOIDCConfig.ClientID,
			GroupsClaim:    expectedOIDCConfig.GroupsClaim,
			IssuerURL:      expectedOIDCConfig.IssuerURL,
			SigningAlgs:    expectedOIDCConfig.SigningAlgs,
			UsernameClaim:  expectedOIDCConfig.UsernameClaim,
			UsernamePrefix: expectedOIDCConfig.UsernamePrefix,
		}
		runtimeStateWithOIDCConfig.CreatedAt = runtimeStateWithOIDCConfig.CreatedAt.Add(time.Hour * 1)

		fixRuntimeStateID2 := "runtimestate2"
		fixOperationID2 := "operation2"
		runtimeStateWithoutOIDCConfig := fixture.FixRuntimeState(fixRuntimeStateID2, fixRuntimeID, fixOperationID2)
		runtimeStateWithoutOIDCConfig.CreatedAt = runtimeStateWithoutOIDCConfig.CreatedAt.Add(time.Hour * 2)

		storage := brokerStorage.RuntimeStates()

		err = storage.Insert(runtimeStateWithOIDCConfig)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithoutOIDCConfig)
		require.NoError(t, err)

		gotRuntimeStates, err := storage.ListByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Len(t, gotRuntimeStates, 2)

		gotRuntimeState, err := storage.GetLatestByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, runtimeStateWithoutOIDCConfig.ID, gotRuntimeState.ID)

		gotRuntimeState, err = storage.GetLatestWithOIDCConfigByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, gotRuntimeState.ID, runtimeStateWithOIDCConfig.ID)
		assert.Equal(t, expectedOIDCConfig, *gotRuntimeState.ClusterConfig.OidcConfig)
	})

	t.Run("should delete runtime states by operation ID", func(t *testing.T) {
		// given
		rs1 := fixture.FixRuntimeState("id1", "rid1", "op-01")
		rs2 := fixture.FixRuntimeState("id2", "rid1", "op-02")
		rs3 := fixture.FixRuntimeState("id3", "rid1", "op-01")

		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
		storage := brokerStorage.RuntimeStates()
		err = storage.Insert(rs1)
		require.NoError(t, err)
		err = storage.Insert(rs2)
		require.NoError(t, err)
		err = storage.Insert(rs3)
		require.NoError(t, err)

		// when
		err = storage.DeleteByOperationID("op-02")
		require.NoError(t, err)

		// then
		rutimeStates, e := storage.ListByRuntimeID("rid1")
		require.NoError(t, e)
		assert.Equal(t, 2, len(rutimeStates))

	})
}
