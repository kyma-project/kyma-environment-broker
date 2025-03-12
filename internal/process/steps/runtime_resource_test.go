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

func TestCheckRuntimeResourceStep(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	t.Run("run when ready", func(t *testing.T) {
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

	t.Run("run when not ready and fail operation", func(t *testing.T) {
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

	t.Run("run when not ready and retry operation", func(t *testing.T) {
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

	t.Run("run when failed and fail operation", func(t *testing.T) {
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
