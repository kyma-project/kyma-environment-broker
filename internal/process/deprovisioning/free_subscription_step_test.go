package deprovisioning

import (
	"context"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const testNamespace = "test-ns"

func TestFreeSubscriptionStep_SubscriptionSecretNameFromInstance(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.GlobalAccountID = operation.GlobalAccountID
	instance.SubscriptionSecretName = subscriptionSecretName

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(newSecretBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
		"tenantName": instance.GlobalAccountID,
	}))
	step := NewFreeSubscriptionStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.SecretBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
}

func TestFreeSubscriptionStep_DoNotReleaseIfShared(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.GlobalAccountID = operation.GlobalAccountID
	instance.SubscriptionSecretName = subscriptionSecretName

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(newSecretBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
		"shared": "true",
	}))
	step := NewFreeSubscriptionStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.SecretBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func TestFreeSubscriptionStep_SubscriptionSecretNameFromTargetSecret(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	operation.GlobalAccountID = instance.GlobalAccountID
	operation.ProvisioningParameters.Parameters.TargetSecret = ptr.String(subscriptionSecretName)
	_ = memoryStorage.Operations().InsertOperation(operation)
	pOperation := fixture.FixProvisioningOperation("provisioning-id", operation.InstanceID)
	pOperation.ProvisioningParameters.Parameters.TargetSecret = ptr.String(subscriptionSecretName)
	_ = memoryStorage.Operations().InsertOperation(pOperation)
	instance.Parameters.Parameters.TargetSecret = ptr.String(subscriptionSecretName)
	instance.SubscriptionSecretName = ""

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newSecretBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID}),
	)
	step := NewFreeSubscriptionStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.SecretBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
}

func TestFreeSubscriptionStep_SubscriptionWasNotAssigned(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	operation.GlobalAccountID = instance.GlobalAccountID
	operation.ProvisioningParameters.Parameters.TargetSecret = nil
	_ = memoryStorage.Operations().InsertOperation(operation)
	pOperation := fixture.FixProvisioningOperation("provisioning-id", operation.InstanceID)
	pOperation.ProvisioningParameters.Parameters.TargetSecret = nil
	_ = memoryStorage.Operations().InsertOperation(pOperation)
	instance.Parameters.Parameters.TargetSecret = nil
	instance.SubscriptionSecretName = ""

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newSecretBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID}),
	)
	step := NewFreeSubscriptionStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
}

func TestReleasingBlocked_ifShootExists(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = subscriptionSecretName
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newSecretBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		newShoot("shoot-01", subscriptionSecretName),
		newShoot("shoot-02", subscriptionSecretName),
	)
	step := NewFreeSubscriptionStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, repeat)

	gotSB, err := gClient.Resource(gardener.SecretBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func newShoot(name, secretBindingName string) *unstructured.Unstructured {
	shoot := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": secretBindingName,
			},
		},
	}
	shoot.SetGroupVersionKind(gardener.ShootGVK)
	return shoot
}

func newSecretBinding(name, secretName string, labels map[string]interface{}) *unstructured.Unstructured {
	secretBinding := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
				"labels":    labels,
			},
			"secretRef": map[string]interface{}{
				"name":      secretName,
				"namespace": testNamespace,
			},
		},
	}
	secretBinding.SetGroupVersionKind(gardener.SecretBindingGVK)
	return secretBinding
}

func fixGCPInstance(instanceID string) internal.Instance {
	instance := fixture.FixInstance(instanceID)
	instance.Provider = pkg.GCP
	return instance
}
func fixDeprovisioningOperationWithPlanID(planID string) internal.Operation {
	deprovisioningOperation := fixture.FixDeprovisioningOperationAsOperation(testOperationID, testInstanceID)
	deprovisioningOperation.ProvisioningParameters.PlanID = planID
	deprovisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = testGlobalAccountID
	deprovisioningOperation.ProvisioningParameters.ErsContext.SubAccountID = testSubAccountID
	return deprovisioningOperation
}
