package deprovisioning

import (
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const kymaNamespace = "kyma-ns"

func TestCheckRuntimeResourceDeletionStep_ResourceNotExists(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	op := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	op.RuntimeResourceName = "runtime-name"
	op.KymaResourceNamespace = kymaNamespace
	memoryStorage := storage.NewMemoryStorage()
	assert.NoError(t, memoryStorage.Operations().InsertOperation(op))
	kcpClient := fake.NewClientBuilder().Build()

	// when
	step := NewCheckRuntimeResourceDeletionStep(memoryStorage, kcpClient, time.Minute)
	_, backoff, err := step.Run(op, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestCheckRuntimeResourceDeletionStep_Run(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	op := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	op.RuntimeResourceName = "runtime-name"
	op.KymaResourceNamespace = kymaNamespace
	memoryStorage := storage.NewMemoryStorage()
	assert.NoError(t, memoryStorage.Operations().InsertOperation(op))
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource(kymaNamespace, "runtime-name")).Build()

	// when
	step := NewCheckRuntimeResourceDeletionStep(memoryStorage, kcpClient, time.Minute)
	_, backoff, err := step.Run(op, fixLogger())

	// then
	assert.NoError(t, err)
	assert.NotZero(t, backoff)
}
