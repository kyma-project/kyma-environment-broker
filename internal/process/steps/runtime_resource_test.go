package steps

import (
	"context"
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const ProvisioningTakesLongerThanUsualForTesting = 20 * time.Second
const ProvisioningTimeoutForTesting = 120 * time.Second

func TestCheckRuntimeResourceStep(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	t.Run("succeed when ready", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("1")
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime(imv1.RuntimeStateReady)
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceStep(os, k8sClient, internal.RetryTuple{Timeout: 2 * time.Second, Interval: time.Second})

		// when
		_, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.Zero(t, backoff)
	})

	t.Run("fail operation when not ready and timeout", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("2")
		operation.CreatedAt = time.Now().Add(-1 * time.Hour)
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		// force immediate timeout
		step := NewCheckRuntimeResourceStep(os, k8sClient, internal.RetryTuple{Timeout: -1 * time.Second, Interval: 2 * time.Second})

		// when
		op, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, domain.Failed, op.State)
	})

	t.Run("retry operation when not ready and not timeout", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("3")
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceStep(os, k8sClient, internal.RetryTuple{Timeout: 2 * time.Second, Interval: time.Second})

		// when
		_, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.NotZero(t, backoff)
	})

	t.Run("fail operation when failed Runtime CR", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("4")
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime(imv1.RuntimeStateFailed)
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceStep(os, k8sClient, internal.RetryTuple{Timeout: 2 * time.Second, Interval: time.Second})

		// when
		op, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, domain.Failed, op.State)
	})
}

func TestCheckRuntimeResourceProvisioningStep(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	t.Run("succeed when ready", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("1")
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime(imv1.RuntimeStateReady)
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: ProvisioningTimeoutForTesting, Interval: time.Second}, ProvisioningTakesLongerThanUsualForTesting)

		// when
		_, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.Zero(t, backoff)
	})

	t.Run("fail operation when not ready and timeout", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("2")
		operation.CreatedAt = time.Now().Add(-1 * time.Hour)
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		// force immediate timeout
		step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: -1 * ProvisioningTimeoutForTesting, Interval: 2 * time.Second}, ProvisioningTakesLongerThanUsualForTesting)

		// when
		op, backoff, _ := step.Run(operation, fixLogger())

		// then
		assert.Zero(t, backoff)
		assert.Equal(t, domain.Failed, op.State)
		// in the real life this resource could be still there, in the step we just trigger deletion (fire-and-forget)
		_, err = GetRuntimeResource(existingRuntime.Name, existingRuntime.Namespace, k8sClient)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("retry operation when not ready and not timeout", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("3")
		operation.Description = "Operation created"

		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: ProvisioningTimeoutForTesting, Interval: time.Second}, ProvisioningTakesLongerThanUsualForTesting)

		// when
		postOperation, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.NotZero(t, backoff)
		assert.Equal(t, "Operation created", postOperation.Description)
		dbOperation, _ := os.GetOperationByID("3")
		assert.Equal(t, "Operation created", dbOperation.Description)
	})

	t.Run("retry operation when not ready and not timeout but operation takes longer than usual", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("4")
		operation.CreatedAt = time.Now().Add(-1*ProvisioningTakesLongerThanUsualForTesting - 20*time.Second)
		operation.Description = "Operation created"
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: ProvisioningTimeoutForTesting, Interval: time.Second}, ProvisioningTakesLongerThanUsualForTesting)

		// when
		postOperation, backoff, err := step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assert.NotZero(t, backoff)
		assert.Equal(t, ProvisioningTakesLongerMessage(ProvisioningTimeoutForTesting), postOperation.Description)

		dbOperation, _ := os.GetOperationByID("4")
		assert.Equal(t, ProvisioningTakesLongerMessage(ProvisioningTimeoutForTesting), dbOperation.Description)
	})
}

func createRuntime(state imv1.State) imv1.Runtime {
	existingRuntime := imv1.Runtime{}
	existingRuntime.ObjectMeta.Name = "runtime-id-000"
	existingRuntime.ObjectMeta.Namespace = "kcp-system"
	existingRuntime.Status.State = state
	condition := v1.Condition{
		Message: "condition message",
	}
	existingRuntime.Status.Conditions = []v1.Condition{condition}
	return existingRuntime
}

func createFakeProvisioningOp(opID string) internal.Operation {
	operation := fixture.FixProvisioningOperation(opID, "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	return operation
}

func TestCheckRuntimeResourceProvisioningStep_P4_CredentialsBindingMarkedDirtyOnTimeout(t *testing.T) {
	// P4 (RED): On provisioning timeout (OperationFailed), the claimed CredentialsBinding
	// must be marked dirty=true.
	// Currently fails because CheckRuntimeResourceProvisioningStep has no gardener client.

	// given
	const cbName = "aws-claimed"
	const gardenerNS = "test"

	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	operation := createFakeProvisioningOp("p4-timeout")
	operation.CreatedAt = time.Now().Add(-1 * time.Hour)
	operation.ProvisioningParameters.Parameters.TargetSecret = strPtr(cbName)
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	existingRuntime := createRuntime("In Progress")
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

	claimedCB := newStepsCredentialsBinding(cbName, gardenerNS, map[string]string{
		"hyperscalerType": "aws",
		"tenantName":      "some-global-account",
	})
	fakeGardenerClient := gardener.NewDynamicFakeClient(claimedCB)

	step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: -1 * ProvisioningTimeoutForTesting, Interval: 2 * time.Second}, ProvisioningTakesLongerThanUsualForTesting)

	// when
	op, backoff, _ := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, domain.Failed, op.State)

	// RED assertion: the claimed CB must be dirty after timeout — currently fails.
	gotCB, err := fakeGardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(gardenerNS).Get(context.Background(), cbName, v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotCB.GetLabels()["dirty"])
}

func TestCheckRuntimeResourceProvisioningStep_P5_CredentialsBindingMarkedDirtyOnRuntimeFailed(t *testing.T) {
	// P5 (RED): When Runtime CR enters Failed state (OperationFailed), the claimed
	// CredentialsBinding must be marked dirty=true.
	// Currently fails because CheckRuntimeResourceProvisioningStep has no gardener client.

	// given
	const cbName = "aws-claimed"
	const gardenerNS = "test"

	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	operation := createFakeProvisioningOp("p5-failed")
	operation.ProvisioningParameters.Parameters.TargetSecret = strPtr(cbName)
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	existingRuntime := createRuntime(imv1.RuntimeStateFailed)
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

	claimedCB := newStepsCredentialsBinding(cbName, gardenerNS, map[string]string{
		"hyperscalerType": "aws",
		"tenantName":      "some-global-account",
	})
	fakeGardenerClient := gardener.NewDynamicFakeClient(claimedCB)

	step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: ProvisioningTimeoutForTesting, Interval: time.Second}, ProvisioningTakesLongerThanUsualForTesting)

	// when
	op, backoff, _ := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, domain.Failed, op.State)

	// RED assertion: the claimed CB must be dirty after Runtime CR failure — currently fails.
	gotCB, err := fakeGardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(gardenerNS).Get(context.Background(), cbName, v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotCB.GetLabels()["dirty"])
}

func strPtr(s string) *string { return &s }

// newStepsCredentialsBinding builds a fake CredentialsBinding for steps package tests.
func newStepsCredentialsBinding(name, namespace string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(labels)
	u.SetGroupVersionKind(gardener.CredentialsBindingGVK)
	return u
}
