package archive

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstanceArchivedFromOperations(t *testing.T) {
	// given
	op1 := fixture.FixProvisioningOperation("op-p-01", "inst01")
	op1.ProvisioningParameters.ErsContext.UserID = "someone@sap.com"
	op2 := fixture.FixDeprovisioningOperationAsOperation("op-p-02", "inst01")

	// when
	archived, err := NewInstanceArchivedFromOperations([]internal.Operation{op1, op2})

	// then
	require.NoError(t, err)
	assert.True(t, op1.CreatedAt.Equal(archived.ProvisioningStartedAt))
	assert.True(t, op1.UpdatedAt.Equal(archived.ProvisioningFinishedAt))
	assert.Equal(t, op1.State, archived.ProvisioningState)

	assert.True(t, op2.CreatedAt.Equal(archived.FirstDeprovisioningStartedAt))
	assert.True(t, op2.UpdatedAt.Equal(archived.FirstDeprovisioningFinishedAt))
	assert.True(t, op2.UpdatedAt.Equal(archived.LastDeprovisioningFinishedAt))

	assert.Equal(t, "inst01", archived.InstanceID)
	assert.Equal(t, op1.ProvisioningParameters.PlanID, archived.PlanID)
	assert.True(t, archived.InternalUser)
}

func TestNewInstanceArchivedFromOperationsNonInternal(t *testing.T) {
	// given
	op1 := fixture.FixProvisioningOperation("op-p-01", "inst01")
	op1.ProvisioningParameters.ErsContext.UserID = "someone@example.org"
	op2 := fixture.FixDeprovisioningOperationAsOperation("op-p-02", "inst01")

	// when
	archived, err := NewInstanceArchivedFromOperations([]internal.Operation{op1, op2})

	// then
	require.NoError(t, err)
	assert.False(t, archived.InternalUser)
}
