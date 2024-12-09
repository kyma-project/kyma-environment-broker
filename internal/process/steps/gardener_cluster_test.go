package steps

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGardenerCluster(t *testing.T) {
	// Given
	g := NewGardenerCluster("gc", "kcp-system")
	err := g.SetKubecofigSecret("kubeconfig-gc", "kcp-system")
	assert.NoError(t, err)
	err = g.SetShootName("c-12345")
	assert.NoError(t, err)

	// When
	d, _ := g.ToYaml()

	// Then
	expectedYaml := `
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: gc
  namespace: kcp-system
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-gc
      namespace: kcp-system
`
	assert.YAMLEq(t, expectedYaml, string(d))
}

func TestSyncGardenerCluster_RunWithExistingResource(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: true,
		DryRun:   false,
	}

	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	err := existingGC.SetShootName("abcd")
	assert.NoError(t, err)
	existingAsUnstructured := existingGC.ToUnstructured()
	existingAsUnstructured.SetLabels(map[string]string{"my-label": "01234"})
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingAsUnstructured).Build()
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	err = os.InsertOperation(operation)
	assert.NoError(t, err)
	svc := NewSyncGardenerCluster(os, k8sClient, kimConfig)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assertGardenerClusterSpec(t, `
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: runtime-id-000
  namespace: kcp-system
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-runtime-id-000
      namespace: kcp-system
`, k8sClient)

	// verify existing label if it still exists
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(GardenerClusterGVK())
	err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(existingAsUnstructured), got)
	assert.NoError(t, err)
	assert.Equal(t, "01234", got.GetLabels()["my-label"])
	gc := NewGardenerClusterFromUnstructured(got)
	assert.Equal(t, "", gc.GetState())
}

func TestSyncGardenerCluster_Run(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: true,
		DryRun:   false,
	}

	k8sClient := fake.NewClientBuilder().Build()
	svc := NewSyncGardenerCluster(os, k8sClient, kimConfig)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	err := os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assertGardenerClusterSpec(t, `
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: runtime-id-000
  namespace: kcp-system
spec:
  shoot:
    name: c-12345
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-runtime-id-000
      namespace: kcp-system
`, k8sClient)
}

func TestCheckGardenerCluster_RunWhenReady(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: true,
		DryRun:   false,
	}
	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	err := existingGC.SetState("Ready")
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingGC.ToUnstructured()).Build()
	step := NewCheckGardenerCluster(os, k8sClient, kimConfig, time.Second)
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

func TestCheckGardenerCluster_RunWhenNotReady_OperationFail(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: true,
		DryRun:   false,
	}
	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	err := existingGC.SetState("In progress")
	assert.NoError(t, err)
	err = existingGC.SetStatusConditions("some condition")
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingGC.ToUnstructured()).Build()
	step := NewCheckGardenerCluster(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	operation.UpdatedAt = time.Now().Add(-1 * time.Hour)
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	op, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Error(t, err)
	assert.Zero(t, backoff)
	assert.Equal(t, domain.Failed, op.State)
}

func TestCheckGardenerCluster_IgnoreWhenNotReadyButKimDrives(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: false,
		DryRun:   false,
	}
	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	err := existingGC.SetState("In progress")
	assert.NoError(t, err)
	err = existingGC.SetStatusConditions("some condition")
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingGC.ToUnstructured()).Build()
	step := NewCheckGardenerCluster(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	operation.UpdatedAt = time.Now().Add(-1 * time.Hour)
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestCheckGardenerCluster_IgnoreWhenNotReadyButKimOnlyPlanUsed(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:      true,
		Plans:        []string{"azure"},
		KimOnlyPlans: []string{"azure"},
		ViewOnly:     true,
		DryRun:       true,
	}
	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	err := existingGC.SetState("In progress")
	assert.NoError(t, err)
	err = existingGC.SetStatusConditions("some condition")
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingGC.ToUnstructured()).Build()
	step := NewCheckGardenerCluster(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	operation.UpdatedAt = time.Now().Add(-1 * time.Hour)
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestCheckGardenerCluster_RunWhenNotReady_Retry(t *testing.T) {
	// given
	os := storage.NewMemoryStorage().Operations()
	kimConfig := broker.KimConfig{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: true,
		DryRun:   false,
	}
	existingGC := NewGardenerCluster("runtime-id-000", "kcp-system")
	err := existingGC.SetState("In progress")
	assert.NoError(t, err)
	err = existingGC.SetStatusConditions("some condition")
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(existingGC.ToUnstructured()).Build()
	step := NewCheckGardenerCluster(os, k8sClient, kimConfig, time.Second)
	operation := fixture.FixProvisioningOperation("op", "instance-id")
	operation.KymaResourceNamespace = "kcp-system"
	operation.RuntimeID = "runtime-id-000"
	operation.ShootName = "c-12345"
	operation.UpdatedAt = time.Now()
	err = os.InsertOperation(operation)
	assert.NoError(t, err)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.NotZero(t, backoff)
}

func assertGardenerClusterSpec(t *testing.T, s string, k8sClient client.Client) {
	scheme := internal.NewSchemeForTests(t)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	expected := unstructured.Unstructured{}
	_, _, err := decoder.Decode([]byte(s), nil, &expected)
	assert.NoError(t, err)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(GardenerClusterGVK())
	err = k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: expected.GetNamespace(),
		Name:      expected.GetName(),
	}, existing)
	assert.NoError(t, err)

	assert.Equal(t, expected.Object["spec"], existing.Object["spec"])
}
