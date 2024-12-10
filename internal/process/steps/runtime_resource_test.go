package steps

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckRuntimeResource_RunWhenReady(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	os := storage.NewMemoryStorage().Operations()
	existingRuntime := createRuntime("Ready")
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()
	kimConfig := fixKimConfigForAzure()

	step := NewCheckRuntimeResourceStep(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestCheckRuntimeResource_RunWhenNotReady_OperationFail(t *testing.T) {
	// given

	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	existingRuntime := createRuntime("In Progress")

	kimConfig := fixKimConfigForAzure()

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()
	step := NewCheckRuntimeResourceStep(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	operation.CreatedAt = time.Now().Add(-1 * time.Hour)
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	op, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Error(t, err)
	assert.Zero(t, backoff)
	assert.Equal(t, domain.Failed, op.State)
}

func TestCheckRuntimeResource_RunWhenNotReady_Retry(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	os := storage.NewMemoryStorage().Operations()

	existingRuntime := createRuntime("In Progress")

	kimConfig := fixKimConfigForAzure()

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&existingRuntime).Build()

	step := NewCheckRuntimeResourceStep(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	operation.CreatedAt = time.Now()
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.NotZero(t, backoff)
}

func fixKimConfigForAzure() broker.KimConfig {
	return broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: false,
	}
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
