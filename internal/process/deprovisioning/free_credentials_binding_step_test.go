package deprovisioning

import (
	"context"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const subscriptionSecretName = "sb-01"
const testNamespace = "test-ns"
const testShootName = "shoot-for-this-instance"

func TestFreeCredentialsBinding_SubscriptionSecretNameFromInstance(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.GlobalAccountID = operation.GlobalAccountID
	instance.SubscriptionSecretName = subscriptionSecretName

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
		"tenantName": instance.GlobalAccountID,
	}))
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
}

func TestFreeCredentialsBinding_DoNotReleaseIfShared(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.GlobalAccountID = operation.GlobalAccountID
	instance.SubscriptionSecretName = subscriptionSecretName

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
		"shared": "true",
	}))
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func TestFreeCredentialsBinding_SubscriptionSecretNameFromTargetSecret(t *testing.T) {
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
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID}),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, _ := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)

	// then
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
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
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
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
	operation.ShootName = testShootName
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = subscriptionSecretName
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		newShoot(testShootName),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	assert.NotZero(t, repeat)

	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func TestFreeCredentialsBinding_MarkedDirtyOnTimeout_ifShootStillExists(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	operation.ShootName = testShootName
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = subscriptionSecretName
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		newShoot(testShootName),
	)
	// negative timeout forces immediate failure on first call
	step := newFreeCredentialsBindingStepWithShootTimeout(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace, -1*time.Second)

	// when
	result, _, _ := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, domain.Failed, result.State)

	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
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

func TestFreeCredentialsBinding_MarkedDirty_WhenOtherInstanceShootExists(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	operation.ShootName = testShootName
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = subscriptionSecretName
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		// Own shoot is gone; only an unrelated shoot exists (no CB reference)
		newShoot("shoot-for-other-instance"),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, err := step.Run(operation, fixLogger())
	assert.Zero(t, backoff)
	assert.NoError(t, err)

	// then — no other shoot references the CB, so mark dirty
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotSB.GetLabels()["dirty"])
}

func TestFreeCredentialsBinding_NotReleased_WhenAnotherShootReferencesCredentialsBinding(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	operation.ShootName = testShootName
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = subscriptionSecretName
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		// Own shoot is gone; another instance's shoot still references the CB
		newShootWithCredentialsBindingRef("shoot-for-other-instance", subscriptionSecretName),
	)
	step := NewFreeCredentialsBindingStep(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then — CB still in use by another instance, skip
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
}

func TestFreeCredentialsBinding_NotMarkedDirtyOnTimeout_ifOtherShootReferencesCredentialsBinding(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.AWSPlanID)
	operation.ShootName = testShootName
	instance := fixGCPInstance(operation.InstanceID)
	instance.SubscriptionSecretName = subscriptionSecretName
	instance.GlobalAccountID = operation.GlobalAccountID

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	gClient := gardener.NewDynamicFakeClient(
		newCredentialsBinding(subscriptionSecretName, "secret-01", map[string]interface{}{
			"tenantName": instance.GlobalAccountID,
		}),
		newShoot(testShootName),
		newShootWithCredentialsBindingRef("shoot-for-other-instance", subscriptionSecretName),
	)
	// negative timeout forces immediate failure on first call
	step := newFreeCredentialsBindingStepWithShootTimeout(memoryStorage.Operations(), memoryStorage.Instances(), gClient, testNamespace, -1*time.Second)

	// when
	result, _, _ := step.Run(operation, fixLogger())

	// then — timed out but CB still used by another instance, do NOT mark dirty
	assert.Equal(t, domain.Failed, result.State)
	gotSB, err := gClient.Resource(gardener.CredentialsBindingResource).Namespace(testNamespace).Get(context.Background(), subscriptionSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, gotSB.GetLabels(), "dirty")
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

func newShoot(name string) *unstructured.Unstructured {
	shoot := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
			},
		},
	}
	shoot.SetGroupVersionKind(gardener.ShootGVK)
	return shoot
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
