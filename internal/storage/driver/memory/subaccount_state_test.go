package memory

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
	subaccountStates := NewSubaccountStates()

	t.Run("should insert and fetch state", func(t *testing.T) {

		err := subaccountStates.UpsertState(givenSubaccount1State1)
		require.NoError(t, err)

		actualStates, err := subaccountStates.ListStates()
		require.NoError(t, err)
		assert.Len(t, actualStates, 1)
		assert.Equal(t, "true", actualStates[0].BetaEnabled)
		assert.Equal(t, "NOT_SET", actualStates[0].UsedForProduction)
	})

	t.Run("should update subaccount state and then fetch it", func(t *testing.T) {

		err := subaccountStates.UpsertState(givenSubaccount1State2)

		require.NoError(t, err)
		actualStates, err := subaccountStates.ListStates()
		require.NoError(t, err)
		assert.Len(t, actualStates, 1)
		assert.Equal(t, "false", actualStates[0].BetaEnabled)
		assert.Equal(t, "NOT_SET", actualStates[0].UsedForProduction)
	})

	t.Run("insert second subaccount state and fetch two states", func(t *testing.T) {

		err := subaccountStates.UpsertState(givenSubaccount2State3)

		require.NoError(t, err)
		actualStates, err := subaccountStates.ListStates()
		require.NoError(t, err)
		assert.Len(t, actualStates, 2)
		expectedStates := []internal.SubaccountState{givenSubaccount1State2, givenSubaccount2State3}

		for _, expectedState := range expectedStates {
			assert.Contains(t, actualStates, expectedState)

		}
	})

	t.Run("delete second subaccount state and fetch one state", func(t *testing.T) {

		err := subaccountStates.DeleteState(subaccountID2)

		require.NoError(t, err)
		actualStates, err := subaccountStates.ListStates()
		require.NoError(t, err)
		assert.Equal(t, givenSubaccount1State2, actualStates[0])
		assert.Len(t, actualStates, 1)
	})

}
