package provisioning

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitProviderValuesStep_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixProvisioningOperation("op-id", "i-id")
	// to be sure Provider ProviderValues is empty
	operation.ProviderValues = internal.ProviderValues{}
	operation.ProvisioningParameters.PlanID = broker.AWSPlanID
	operation.ProvisioningParameters.Parameters.Region = ptr.String("eu-central-1")

	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewInitProviderValuesStep(memoryStorage.Operations(), input.Config{
		DefaultGardenerShootPurpose:  "production",
		TrialNodesNumber:             1,
		DefaultTrialProvider:         "aws",
		MultiZoneCluster:             false,
		ControlPlaneFailureTolerance: "node",
	}, nil, false)

	// when
	gotOperation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.Zero(t, repeat)
	assert.Equal(t, gotOperation.ProviderValues.ProviderType, "aws")
	assert.Equal(t, gotOperation.ProviderValues.Region, "eu-central-1")
	assert.Equal(t, gotOperation.ProviderValues.Purpose, "production")
	assert.Equal(t, gotOperation.ProviderValues.FailureTolerance, ptr.String("node"))

	storedOperation, err := memoryStorage.Operations().GetProvisioningOperationByID(operation.ID)
	require.NoError(t, err)
	assert.Equal(t, storedOperation.ProviderValues.ProviderType, "aws")
	assert.Equal(t, storedOperation.ProviderValues.Region, "eu-central-1")
	assert.Equal(t, storedOperation.ProviderValues.Purpose, "production")
	assert.Equal(t, storedOperation.ProviderValues.FailureTolerance, ptr.String("node"))
}
