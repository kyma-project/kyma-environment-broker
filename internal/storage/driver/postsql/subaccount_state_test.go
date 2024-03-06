package postsql_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	subaccountID1 = "subaccountID1"
	subaccountID2 = "subaccountID2"
)

var (
	givenSubaccount1State1 = internal.SubaccountState{
		ID:                subaccountID1,
		BetaEnabled:       "true",
		UsedForProduction: "NOT_SET",
		ModifiedAt:        10,
	}

	givenSubaccount1State2 = internal.SubaccountState{
		ID:                subaccountID1,
		BetaEnabled:       "false",
		UsedForProduction: "NOT_SET",
		ModifiedAt:        54,
	}
	givenSubaccount2State3 = internal.SubaccountState{
		ID:                subaccountID2,
		BetaEnabled:       "true",
		UsedForProduction: "USED_FOR_PRODUCTION",
		ModifiedAt:        108,
	}
)

func TestSubaccountState(t *testing.T) {
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
	require.NoError(t, err)
	require.NotNil(t, brokerStorage)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	svc := brokerStorage.SubaccountStates()

	t.Run("should insert and fetch state", func(t *testing.T) {

		err = svc.UpsertState(givenSubaccount1State1)
		require.NoError(t, err)

		subaccountStates, err := svc.ListStates()
		require.NoError(t, err)
		assert.Len(t, subaccountStates, 1)
		assert.Equal(t, "true", subaccountStates[0].BetaEnabled)
		assert.Equal(t, "NOT_SET", subaccountStates[0].UsedForProduction)
	})

	t.Run("should update subaccount state and then fetch it", func(t *testing.T) {

		err = svc.UpsertState(givenSubaccount1State2)

		require.NoError(t, err)
		subaccountStates, err := svc.ListStates()
		require.NoError(t, err)
		assert.Len(t, subaccountStates, 1)
		assert.Equal(t, "false", subaccountStates[0].BetaEnabled)
		assert.Equal(t, "NOT_SET", subaccountStates[0].UsedForProduction)
	})

	t.Run("insert second subaccount state and fetch two states", func(t *testing.T) {

		err = svc.UpsertState(givenSubaccount2State3)

		require.NoError(t, err)
		subaccountStates, err := svc.ListStates()
		require.NoError(t, err)
		assert.Len(t, subaccountStates, 2)
		expectedStates := []internal.SubaccountState{givenSubaccount1State2, givenSubaccount2State3}

		for _, expectedState := range expectedStates {
			assert.Contains(t, subaccountStates, expectedState)

		}
	})

	t.Run("delete second subaccount state and fetch one state", func(t *testing.T) {

		err = svc.DeleteState(subaccountID2)

		require.NoError(t, err)
		subaccountStates, err := svc.ListStates()
		require.NoError(t, err)
		assert.Equal(t, givenSubaccount1State2, subaccountStates[0])
		assert.Len(t, subaccountStates, 1)
	})

}
