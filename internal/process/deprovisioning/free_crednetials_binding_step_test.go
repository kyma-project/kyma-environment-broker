package deprovisioning

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFreeCredentialsBinding_SubscriptionSecretNameFromInstance(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.GlobalAccountID = operation.GlobalAccountID
	instance.SubscriptionSecretName = "sb-01"

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(newCredentialsBinding("sb-01", "secret-01", map[string]interface{}{
		"tenantName": instance.GlobalAccountID,
	}))
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), "sb-01", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
}

func TestFreeCredentialsBinding_DoNotReleaseIfShared(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.GlobalAccountID = operation.GlobalAccountID
	instance.SubscriptionSecretName = "sb-01"

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(newCredentialsBinding("sb-01", "secret-01", map[string]interface{}{
		"shared": "true",
	}))
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), "sb-01", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func TestFreeCredentialsBinding_SubscriptionSecretNameFromTargetSecret(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	operation.GlobalAccountID = instance.GlobalAccountID
	operation.ProvisioningParameters.Parameters.TargetSecret = ptr.String("sb-01")
	_ = memoryStorage.Operations().InsertOperation(operation)
	pOperation := fixture.FixProvisioningOperation("provisioning-id", operation.InstanceID)
	pOperation.ProvisioningParameters.Parameters.TargetSecret = ptr.String("sb-01")
	_ = memoryStorage.Operations().InsertOperation(pOperation)
	instance.Parameters.Parameters.TargetSecret = ptr.String("sb-01")
	instance.SubscriptionSecretName = ""

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding("sb-01", "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID}),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), "sb-01", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
}

func TestFreeCredentialsBinding_SubscriptionWasNotAssigned(t *testing.T) {
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
		newCredentialsBinding("sb-01", "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID}),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
}

func TestFreeCredentialsBinding_ReleasingBlocked_ifShootExists(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = "sb-01"
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding("sb-01", "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		newShootWithCredentialsBindingRef("shoot-01", "sb-01"),
		newShootWithCredentialsBindingRef("shoot-02", "sb-01"),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, repeat)

	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), "sb-01", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func newCredentialsBinding(name, secretName string, labels map[string]interface{}) *unstructured.Unstructured {
	secretBinding := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
				"labels":    labels,
			},
			"credentialsRef": map[string]interface{}{
				"name":      secretName,
				"namespace": testNamespace,
			},
		},
	}
	secretBinding.SetGroupVersionKind(gardener.CredentialsBindingGVK)
	return secretBinding
}

func newShootWithCredentialsBindingRef(name, credentialsBindingName string) *unstructured.Unstructured {
	shoot := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"credentialsBindingName": credentialsBindingName,
			},
		},
	}
	shoot.SetGroupVersionKind(gardener.ShootGVK)
	return shoot
}
