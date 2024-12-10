package deprovisioning

import (
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const kymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma
  namespace: kyma-system
spec:
  sync:
    strategy: secret
  channel: stable
  modules: []
`

func TestDeleteKymaResource_HappyFlow(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), kcpClient, fakeConfigProvider{})
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.Contains(t, err.Error(), fmt.Sprintf("instance operation with id %s already exist", fixOperationID))

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
}

func TestDeleteKymaResource_EmptyRuntimeIDAndKymaResourceName(t *testing.T) {
	// Given
	operation := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	operation.KymaResourceNamespace = "kyma-system"
	operation.RuntimeID = ""
	operation.KymaResourceName = ""
	instance := fixture.FixInstance(fixInstanceID)

	kcpClient := fake.NewClientBuilder().Build()
	memoryStorage := storage.NewMemoryStorage()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDeleteKymaResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), kcpClient, fakeConfigProvider{})
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.Contains(t, err.Error(), fmt.Sprintf("instance operation with id %s already exist", fixOperationID))
	err = memoryStorage.Instances().Insert(instance)
	require.NoError(t, err)

	// When
	_, backoff, err := step.Run(operation, fixLogger())

	// Then
	assert.Zero(t, backoff)
}

type fakeConfigProvider struct {
}

func (fakeConfigProvider) ProvideForGivenPlan(_ string) (*internal.ConfigForPlan, error) {
	return &internal.ConfigForPlan{
		KymaTemplate: kymaTemplate,
	}, nil
}
