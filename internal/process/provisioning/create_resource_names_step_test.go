package provisioning

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

const (
	shootName       = "c-1234567"
	instanceID      = "58f8c703-1756-48ab-9299-a847974d1fee"
	operationID     = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	globalAccountID = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
	subAccountID    = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
	runtimeID       = "2498c8ee-803a-43c2-8194-6d6dd0354c30"
)

func TestCreateResourceNamesStep_HappyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixProvisioningOperationWithEmptyResourceName()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateResourceNamesStep(memoryStorage.Operations())

	// when
	postOperation, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
	assert.Equal(t, operation.RuntimeID, postOperation.RuntimeID)
	assert.Equal(t, postOperation.KymaResourceName, operation.RuntimeID)
	assert.Equal(t, postOperation.KymaResourceNamespace, "kyma-system")
	assert.Equal(t, postOperation.RuntimeResourceName, operation.RuntimeID)
	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func fixProvisioningOperationWithEmptyResourceName() internal.Operation {
	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.KymaResourceName = ""
	operation.RuntimeResourceName = ""
	return operation
}

func TestCreateResourceNamesStep_NoRuntimeID(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixProvisioningOperationWithEmptyResourceName()
	operation.RuntimeID = ""

	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateResourceNamesStep(memoryStorage.Operations())

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.ErrorContains(t, err, "RuntimeID not set")
	assert.Zero(t, backoff)
}

func fixInstance() internal.Instance {
	instance := fixture.FixInstance(instanceID)
	instance.GlobalAccountID = globalAccountID

	return instance
}

func fixProvisioningParametersWithPlanID(planID, region string, platformRegion string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:    planID,
		ServiceID: "",
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
		},
		PlatformRegion: platformRegion,
		Parameters: pkg.ProvisioningParametersDTO{
			Region: ptr.String(region),
			Name:   "dummy",
			Zones:  []string{"europe-west3-b", "europe-west3-c"},
		},
	}
}
