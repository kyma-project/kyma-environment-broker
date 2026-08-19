package steps

import (
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	t.Run("set RuntimeResourceCreatedAt on first execution", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("5")
		assert.Nil(t, operation.RuntimeResourceCreatedAt)
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
		assert.NotNil(t, postOperation.RuntimeResourceCreatedAt)

		dbOperation, _ := os.GetOperationByID("5")
		assert.NotNil(t, dbOperation.RuntimeResourceCreatedAt)
	})

	t.Run("preserve RuntimeResourceCreatedAt across retries", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("6")
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: ProvisioningTimeoutForTesting, Interval: time.Second}, ProvisioningTakesLongerThanUsualForTesting)

		// when - first execution
		postOperation1, _, _ := step.Run(operation, fixLogger())
		firstTimestamp := postOperation1.RuntimeResourceCreatedAt
		assert.NotNil(t, firstTimestamp)

		time.Sleep(10 * time.Millisecond) // ensure time has passed

		// when - second execution (retry)
		postOperation2, backoff, err := step.Run(postOperation1, fixLogger())

		// then
		assert.NoError(t, err)
		assert.NotZero(t, backoff)
		assert.NotNil(t, postOperation2.RuntimeResourceCreatedAt)
		assert.True(t, firstTimestamp.Equal(*postOperation2.RuntimeResourceCreatedAt), "RuntimeResourceCreatedAt should not change on retry")
	})

	t.Run("timeout calculated from RuntimeResourceCreatedAt not operation.CreatedAt", func(t *testing.T) {
		// given
		operation := createFakeProvisioningOp("7")
		// Simulate operation that was in queue for 2 hours before step executed
		operation.CreatedAt = time.Now().Add(-2 * time.Hour)
		err = os.InsertOperation(operation)
		assert.NoError(t, err)

		existingRuntime := createRuntime("In Progress")
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

		// Set timeout to 30 seconds - if calculated from CreatedAt, it would immediately fail
		step := NewCheckRuntimeResourceProvisioningStep(os, k8sClient, internal.RetryTuple{Timeout: 30 * time.Second, Interval: time.Second}, ProvisioningTakesLongerThanUsualForTesting)

		// when - first execution sets RuntimeResourceCreatedAt to now
		postOperation, backoff, err := step.Run(operation, fixLogger())

		// then - should NOT timeout because RuntimeResourceCreatedAt is recent
		assert.NoError(t, err)
		assert.NotZero(t, backoff, "Should retry, not timeout")
		assert.NotEqual(t, domain.Failed, postOperation.State, "Should NOT fail due to timeout")
		assert.NotNil(t, postOperation.RuntimeResourceCreatedAt)
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
