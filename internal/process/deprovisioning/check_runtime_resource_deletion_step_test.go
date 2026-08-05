package deprovisioning

import (
	"context"
	"testing"
	"time"

	kgardener "github.com/kyma-project/kyma-environment-broker/common/gardener"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestCheckRuntimeResourceDeletionStep_D1_CredentialsBindingMarkedDirtyOnTimeout(t *testing.T) {
	// D1 (RED): When CheckRuntimeResourceDeletionStep times out (OperationFailed), the
	// claimed CredentialsBinding must be marked dirty=true.
	// Currently fails because CheckRuntimeResourceDeletionStep has no gardener client.

	// given
	const cbName = "aws-claimed"
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	memoryStorage := storage.NewMemoryStorage()

	// Instance with claimed CB
	instance := fixture.FixInstance(fixInstanceID)
	instance.SubscriptionSecretName = cbName
	assert.NoError(t, memoryStorage.Instances().Insert(instance))

	op := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	op.RuntimeResourceName = "runtime-name"
	op.KymaResourceNamespace = kymaNamespace
	op.CreatedAt = op.CreatedAt.Add(-2 * time.Hour) // force timeout
	assert.NoError(t, memoryStorage.Operations().InsertOperation(op))

	// Runtime still exists → step keeps retrying until timeout
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource(kymaNamespace, "runtime-name")).Build()

	// Claimed CB (tenantName set, not shared)
	claimedCB := newDeletionCheckCBHelper(cbName, testNamespace, map[string]string{
		"hyperscalerType": "aws",
		"tenantName":      "some-ga",
	})
	fakeGardenerClient := kgardener.NewDynamicFakeClient(claimedCB)

	// Negative timeout forces immediate failure
	step := NewCheckRuntimeResourceDeletionStep(memoryStorage, kcpClient, -1*time.Second)

	// when
	result, backoff, _ := step.Run(op, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, domain.Failed, result.State)

	// RED assertion: claimed CB must be dirty after timeout — currently fails.
	gotCB, err := fakeGardenerClient.Resource(kgardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), cbName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotCB.GetLabels()["dirty"])
}

func newDeletionCheckCBHelper(name, namespace string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(labels)
	u.SetGroupVersionKind(kgardener.CredentialsBindingGVK)
	return u
}
