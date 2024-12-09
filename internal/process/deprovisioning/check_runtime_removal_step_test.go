package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRuntimeRemovalStep(t *testing.T) {
	for _, tc := range []struct {
		givenState       gqlschema.OperationState
		expectedDuration bool
	}{
		{givenState: gqlschema.OperationStatePending, expectedDuration: true},
		{givenState: gqlschema.OperationStateInProgress, expectedDuration: true},
		{givenState: gqlschema.OperationStateSucceeded, expectedDuration: false},
	} {
		t.Run(string(tc.givenState), func(t *testing.T) {
			// given
			memoryStorage := storage.NewMemoryStorage()
			provisionerClient := provisioner.NewFakeClient()
			svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, time.Minute)
			dOp := fixDeprovisioningOperation().Operation
			err := memoryStorage.Instances().Insert(internal.Instance{
				GlobalAccountID: "global-acc",
				InstanceID:      dOp.InstanceID,
			})
			require.NoError(t, err)
			provisionerOp, _ := provisionerClient.DeprovisionRuntime(dOp.GlobalAccountID, dOp.RuntimeID)
			provisionerClient.FinishProvisionerOperation(provisionerOp, tc.givenState)
			dOp.ProvisionerOperationID = provisionerOp

			// when
			_, d, err := svc.Run(dOp, fixLogger())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDuration, d > 0)
		})
	}
}

func TestCheckRuntimeRemovalStep_ProvisionerFailed(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	provisionerClient := provisioner.NewFakeClient()
	svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, time.Minute)
	dOp := fixDeprovisioningOperation().Operation
	err := memoryStorage.Operations().InsertOperation(dOp)
	require.NoError(t, err)
	err = memoryStorage.Instances().Insert(internal.Instance{
		GlobalAccountID: "global-acc",
		InstanceID:      dOp.InstanceID,
	})
	require.NoError(t, err)
	provisionerOp, _ := provisionerClient.DeprovisionRuntime(dOp.GlobalAccountID, dOp.RuntimeID)
	provisionerClient.FinishProvisionerOperation(provisionerOp, gqlschema.OperationStateFailed)
	dOp.ProvisionerOperationID = provisionerOp

	// when
	op, _, err := svc.Run(dOp, fixLogger())

	// then
	require.Error(t, err)
	assert.Equal(t, domain.Failed, op.State)
}

func TestCheckRuntimeRemovalStep_InstanceDeleted(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	provisionerClient := provisioner.NewFakeClient()
	svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, time.Minute)
	dOp := fixDeprovisioningOperation().Operation
	err := memoryStorage.Operations().InsertOperation(dOp)
	require.NoError(t, err)

	// when
	_, backoff, err := svc.Run(dOp, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestCheckRuntimeRemovalStep_NoProvisionerOperationID(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, time.Minute)
	dOp := fixDeprovisioningOperation().Operation
	err := memoryStorage.Instances().Insert(internal.Instance{
		GlobalAccountID: "global-acc",
		InstanceID:      dOp.InstanceID,
	})
	require.NoError(t, err)
	dOp.ProvisionerOperationID = ""

	// when
	_, d, err := svc.Run(dOp, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
}
