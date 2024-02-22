package postsql_test

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
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

	t.Run("should insert and fetch RuntimeState with Reconciler input", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		fixRuntimeStateID := uuid.NewString()
		fixRuntimeID := "runtimeID"
		fixOperationID := "operationID"
		givenRuntimeState := fixture.FixRuntimeState(fixRuntimeStateID, fixRuntimeID, fixOperationID)
		fixClusterSetup := fixture.FixClusterSetup(fixRuntimeID)
		givenRuntimeState.ClusterSetup = &fixClusterSetup

		storage := brokerStorage.RuntimeStates()

		err = storage.Insert(givenRuntimeState)
		require.NoError(t, err)

		runtimeStates, err := storage.ListByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Len(t, runtimeStates, 1)
		assert.Equal(t, fixRuntimeStateID, runtimeStates[0].ID)
		assert.Equal(t, fixRuntimeID, runtimeStates[0].ClusterSetup.RuntimeID)

		const kymaVersion = "2.0.0"
		assert.Equal(t, kymaVersion, runtimeStates[0].ClusterSetup.KymaConfig.Version)

		state, err := storage.GetByOperationID(fixOperationID)
		require.NoError(t, err)
		assert.Equal(t, fixRuntimeStateID, state.ID)
		assert.Equal(t, fixRuntimeID, state.ClusterSetup.RuntimeID)
	})

	t.Run("should distinguish between latest RuntimeStates with and without Reconciler input", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		fixRuntimeID := "runtimeID"

		fixRuntimeStateID1 := "runtimestate1"
		fixOperationID1 := "operation1"
		runtimeStateWithoutReconcilerInput1 := fixture.FixRuntimeState(fixRuntimeStateID1, fixRuntimeID, fixOperationID1)
		runtimeStateWithoutReconcilerInput1.CreatedAt = runtimeStateWithoutReconcilerInput1.CreatedAt.Add(time.Hour * 2)

		fixRuntimeStateID2 := "runtimestate2"
		fixOperationID2 := "operation2"
		runtimeStateWithReconcilerInput1 := fixture.FixRuntimeState(fixRuntimeStateID2, fixRuntimeID, fixOperationID2)
		runtimeStateWithReconcilerInput1.CreatedAt = runtimeStateWithReconcilerInput1.CreatedAt.Add(time.Hour * 1)
		runtimeStateWithReconcilerInput1.ClusterSetup = &reconcilerApi.Cluster{
			RuntimeID: fixRuntimeID,
		}

		fixRuntimeStateID3 := "runtimestate3"
		fixOperationID3 := "operation3"
		runtimeStateWithoutReconcilerInput2 := fixture.FixRuntimeState(fixRuntimeStateID3, fixRuntimeID, fixOperationID3)

		fixRuntimeStateID4 := "runtimestate4"
		fixOperationID4 := "operation4"
		runtimeStateWithReconcilerInput2 := fixture.FixRuntimeState(fixRuntimeStateID4, fixRuntimeID, fixOperationID4)
		runtimeStateWithReconcilerInput2.ClusterSetup = &reconcilerApi.Cluster{
			RuntimeID: fixRuntimeID,
		}

		storage := brokerStorage.RuntimeStates()

		err = storage.Insert(runtimeStateWithoutReconcilerInput1)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithReconcilerInput1)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithoutReconcilerInput2)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithReconcilerInput2)
		require.NoError(t, err)

		gotRuntimeStates, err := storage.ListByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Len(t, gotRuntimeStates, 4)

		gotRuntimeState, err := storage.GetLatestByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, gotRuntimeState.ID, runtimeStateWithoutReconcilerInput1.ID)
		assert.Nil(t, gotRuntimeState.ClusterSetup)

		gotRuntimeState, err = storage.GetLatestWithReconcilerInputByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, gotRuntimeState.ID, runtimeStateWithReconcilerInput1.ID)
		assert.NotNil(t, gotRuntimeState.ClusterSetup)
		assert.Equal(t, gotRuntimeState.ClusterSetup.RuntimeID, runtimeStateWithReconcilerInput1.ClusterSetup.RuntimeID)
	})

	t.Run("should fetch latest RuntimeState with Kyma version", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		fixRuntimeID := "runtimeID"
		fixKymaVersion := "2.0.3"

		fixRuntimeStateID1 := "runtimestate1"
		fixOperationID1 := "operation1"
		runtimeStateWithoutReconcilerInput := fixture.FixRuntimeState(fixRuntimeStateID1, fixRuntimeID, fixOperationID1)
		runtimeStateWithoutReconcilerInput.CreatedAt = runtimeStateWithoutReconcilerInput.CreatedAt.Add(time.Hour * 2)

		fixRuntimeStateID2 := "runtimestate2"
		fixOperationID2 := "operation2"
		runtimeStateWithReconcilerInput := fixture.FixRuntimeState(fixRuntimeStateID2, fixRuntimeID, fixOperationID2)
		runtimeStateWithReconcilerInput.CreatedAt = runtimeStateWithReconcilerInput.CreatedAt.Add(time.Hour * 1)
		runtimeStateWithReconcilerInput.ClusterSetup = &reconcilerApi.Cluster{
			KymaConfig: reconcilerApi.KymaConfig{
				Version: fixKymaVersion,
			},
			RuntimeID: fixRuntimeID,
		}

		// runtimeStateWithoutVersion := fixture.FixRuntimeState("fixRuntimeStateID3", fixRuntimeID, fixOperationID2)
		runtimeStateWithoutVersion := internal.NewRuntimeState(fixRuntimeID, fixOperationID2, nil, &gqlschema.GardenerConfigInput{})
		runtimeStateWithoutVersion.ID = "fixRuntimeStateID3"
		runtimeStateWithoutVersion.CreatedAt = runtimeStateWithReconcilerInput.CreatedAt.Add(time.Hour * 3)

		storage := brokerStorage.RuntimeStates()

		err = storage.Insert(runtimeStateWithoutReconcilerInput)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithReconcilerInput)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithoutVersion)
		require.NoError(t, err)

		gotRuntimeStates, err := storage.ListByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Len(t, gotRuntimeStates, 3)

		gotRuntimeState, err := storage.GetLatestByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, runtimeStateWithoutVersion.ID, gotRuntimeState.ID)
		assert.Nil(t, gotRuntimeState.ClusterSetup)

		gotRuntimeState, err = storage.GetLatestWithKymaVersionByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, gotRuntimeState.ID, runtimeStateWithReconcilerInput.ID)
		assert.NotNil(t, gotRuntimeState.ClusterSetup)
		assert.Equal(t, fixKymaVersion, gotRuntimeState.ClusterSetup.KymaConfig.Version)
	})

	t.Run("should fetch latest RuntimeState with Kyma version stored only in the kyma_version field", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		fixRuntimeID := "runtimeID"
		fixKymaVersion := "2.0.3"

		fixRuntimeStateID1 := "runtimestate1"
		fixOperationID1 := "operation1"
		runtimeStateWithoutReconcilerInput := fixture.FixRuntimeState(fixRuntimeStateID1, fixRuntimeID, fixOperationID1)
		runtimeStateWithoutReconcilerInput.CreatedAt = runtimeStateWithoutReconcilerInput.CreatedAt.Add(time.Hour * 2)

		fixRuntimeStateID2 := "runtimestate2"
		fixOperationID2 := "operation2"
		runtimeStateWithReconcilerInput := fixture.FixRuntimeState(fixRuntimeStateID2, fixRuntimeID, fixOperationID2)
		runtimeStateWithReconcilerInput.CreatedAt = runtimeStateWithReconcilerInput.CreatedAt.Add(time.Hour * 1)
		runtimeStateWithReconcilerInput.ClusterSetup = &reconcilerApi.Cluster{
			KymaConfig: reconcilerApi.KymaConfig{
				Version: fixKymaVersion,
			},
			RuntimeID: fixRuntimeID,
		}

		runtimeStatePlainVersion := internal.NewRuntimeState(fixRuntimeID, fixOperationID2, nil, &gqlschema.GardenerConfigInput{})
		runtimeStatePlainVersion.ID = "fixRuntimeStateID3"
		runtimeStatePlainVersion.CreatedAt = runtimeStateWithReconcilerInput.CreatedAt.Add(time.Hour * 3)
		runtimeStatePlainVersion.KymaVersion = "2.1.55"

		storage := brokerStorage.RuntimeStates()

		err = storage.Insert(runtimeStateWithoutReconcilerInput)
		require.NoError(t, err)
		err = storage.Insert(runtimeStateWithReconcilerInput)
		require.NoError(t, err)
		err = storage.Insert(runtimeStatePlainVersion)
		require.NoError(t, err)

		gotRuntimeStates, err := storage.ListByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Len(t, gotRuntimeStates, 3)

		gotRuntimeState, err := storage.GetLatestWithKymaVersionByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, runtimeStatePlainVersion.ID, gotRuntimeState.ID)
		assert.Equal(t, "2.1.55", gotRuntimeState.GetKymaVersion())
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
		fixKymaVersion := "2.0.4"
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
		runtimeStateWithoutOIDCConfig.ClusterSetup = &reconcilerApi.Cluster{
			KymaConfig: reconcilerApi.KymaConfig{
				Version: fixKymaVersion,
			},
			RuntimeID: fixRuntimeID,
		}

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
		assert.NotNil(t, gotRuntimeState.ClusterSetup)

		gotRuntimeState, err = storage.GetLatestWithOIDCConfigByRuntimeID(fixRuntimeID)
		require.NoError(t, err)
		assert.Equal(t, gotRuntimeState.ID, runtimeStateWithOIDCConfig.ID)
		assert.Nil(t, gotRuntimeState.ClusterSetup)
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
