package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitProviderValuesStep_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixture.FixProvisioningOperation("op-id", "i-id")
	// to be sure Provider ProviderValues is empty
	operation.ProvisioningParameters.PlanID = broker.AWSPlanID
	operation.ProvisioningParameters.Parameters.Region = ptr.String("eu-central-1")

	instance := fixture.FixInstance("i-id")
	instance.Provider = ""
	err := memoryStorage.Instances().Insert(instance)
	require.NoError(t, err)
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewInitProviderValuesStep(memoryStorage.Operations(), memoryStorage.Instances(), broker.InfrastructureManager{
		DefaultGardenerShootPurpose:  "production",
		DefaultTrialProvider:         "aws",
		MultiZoneCluster:             false,
		ControlPlaneFailureTolerance: "node",
	}, nil)

	// when
	gotOperation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.Zero(t, repeat)
	assert.Equal(t, "aws", gotOperation.ProviderValues.ProviderType)
	assert.Equal(t, "eu-central-1", gotOperation.ProviderValues.Region)
	assert.Equal(t, "production", gotOperation.ProviderValues.Purpose)
	assert.Equal(t, ptr.String("node"), gotOperation.ProviderValues.FailureTolerance)

	storedOperation, err := memoryStorage.Operations().GetProvisioningOperationByID(operation.ID)
	require.NoError(t, err)
	assert.Equal(t, storedOperation.ProviderValues.ProviderType, "aws")
	assert.Equal(t, storedOperation.ProviderValues.Region, "eu-central-1")
	assert.Equal(t, storedOperation.ProviderValues.Purpose, "production")
	assert.Equal(t, storedOperation.ProviderValues.FailureTolerance, ptr.String("node"))

	gotInstance, err := memoryStorage.Instances().GetByID("i-id")
	require.NoError(t, err)
	assert.Equal(t, runtime.CloudProvider("AWS"), gotInstance.Provider)

}
