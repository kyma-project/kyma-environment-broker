package provisioning

import (
	"context"
	"reflect"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"

	btpoperatorcredentials "github.com/kyma-project/kyma-environment-broker/internal/btpmanager/credentials"

	kgardener "github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apicorev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestInjectBTPOperatorCredentialsStep(t *testing.T) {
	t.Run("should execute step flawlessly", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)
		assert.NoError(t, err)

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		operation := fixProvisioningOperationWithClusterIDAndCredentials(k8sClient)
		expectedSecretData := createExpectedSecretData(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.ServiceManagerClusterID)

		step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient), kgardener.NewClient(kgardener.NewDynamicFakeClient(), ""))

		// when
		_, _, err = step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assertTheNamespaceIsPresent(t, k8sClient)
		assertTheSecretIsAsExpected(t, k8sClient, expectedSecretData)

		// when
		operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = "rotated-sample-client-secret"
		expectedRotatedSecretData := createExpectedSecretData(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.ServiceManagerClusterID)
		_, _, err = step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assertTheSecretIsAsExpected(t, k8sClient, expectedRotatedSecretData)
	})
	t.Run("should fail when RuntimeID is empty", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)
		assert.NoError(t, err)

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.RuntimeID = ""

		step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient), kgardener.NewClient(kgardener.NewDynamicFakeClient(), ""))

		// when
		processedOperation, _, _ := step.Run(operation, fixLogger())

		// then
		assert.Equal(t, domain.Failed, processedOperation.State)
	})
}

func TestInjectBTPOperatorCredentialsWhenSecretAlreadyExistsStep(t *testing.T) {
	t.Run("should overwrite secret created by user", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		userSecret := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      "sap-btp-manager",
				"namespace": "kyma-system",
			},
		}}

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)
		assert.NoError(t, err)

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		err = k8sClient.Create(context.TODO(), userSecret)

		require.NoError(t, err)

		operation := fixProvisioningOperationWithClusterIDAndCredentials(k8sClient)
		expectedSecretData := createExpectedSecretData(operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.ServiceManagerClusterID)

		step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient), kgardener.NewClient(kgardener.NewDynamicFakeClient(), ""))

		// when
		_, _, err = step.Run(operation, fixLogger())

		// then
		assert.NoError(t, err)
		assertTheSecretIsAsExpected(t, k8sClient, expectedSecretData)
	})
}

func fixProvisioningOperationWithClusterIDAndCredentials(k8sClient client.WithWatch) internal.Operation {
	operation := fixProvisioningOperationWithCredentials()
	operation.InstanceDetails.ServiceManagerClusterID = "cluster-id"
	return operation
}

func fixProvisioningOperationWithCredentials() internal.Operation {
	operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientID:          "sample-client-id",
		ClientSecret:      "sample-client-secret",
		ServiceManagerURL: "www.service.manager.url.com",
		URL:               "www.sample.url.com",
		XSAppName:         "sample-app-name",
	}
	return operation
}

func assertTheSecretIsAsExpected(t *testing.T, k8sClient client.WithWatch, expected map[string][]byte) {
	secretFromCluster := apicorev1.Secret{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: btpoperatorcredentials.BtpManagerSecretNamespace, Name: btpoperatorcredentials.BtpManagerSecretName}, &secretFromCluster)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(expected, secretFromCluster.Data))
	assert.True(t, reflect.DeepEqual(btpoperatorcredentials.BtpManagerLabels, secretFromCluster.Labels))
	assert.True(t, reflect.DeepEqual(btpoperatorcredentials.BtpManagerAnnotations, secretFromCluster.Annotations))
}

func assertTheNamespaceIsPresent(t *testing.T, k8sClient client.WithWatch) {
	namespace := apicorev1.Namespace{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: btpoperatorcredentials.BtpManagerSecretNamespace}, &namespace)
	require.NoError(t, err)
}

func createExpectedSecretData(credentials *internal.ServiceManagerOperatorCredentials, clusterID string) map[string][]byte {
	return map[string][]byte{
		"clientid":     []byte(credentials.ClientID),
		"clientsecret": []byte(credentials.ClientSecret),
		"sm_url":       []byte(credentials.ServiceManagerURL),
		"tokenurl":     []byte(credentials.URL),
		"cluster_id":   []byte(clusterID),
	}
}

func TestInjectBTPOperatorCredentialsStep_P6_CredentialsBindingMarkedDirtyOnOperationFailed(t *testing.T) {
	// P6 (RED): When InjectBTPOperatorCredentialsStep calls OperationFailed (empty RuntimeID),
	// the claimed CredentialsBinding must be marked dirty=true.
	// Currently fails because the step has no gardener client.

	// given
	const cbName = "aws-claimed"
	const gardenerNS = "test"

	memoryStorage := storage.NewMemoryStorage()

	schm := internal.NewSchemeForTests(t)
	err := apiextensionsv1.AddToScheme(schm)
	assert.NoError(t, err)

	k8sClient := fake.NewClientBuilder().WithScheme(schm).Build()

	operation := fixture.FixProvisioningOperation("op-p6", "inst-p6")
	operation.RuntimeID = ""
	operation.ProvisioningParameters.Parameters.TargetSecret = strPtr(cbName)

	// Claimed CB
	claimedCB := newInjectBTPCBHelper(cbName, gardenerNS, map[string]string{
		"hyperscalerType": "aws",
		"tenantName":      "some-ga",
	})
	fakeGardenerClient := kgardener.NewDynamicFakeClient(claimedCB)

	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient), kgardener.NewClient(fakeGardenerClient, gardenerNS))

	// when
	op, _, _ := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, domain.Failed, op.State)

	// RED assertion: claimed CB must be dirty after OperationFailed — currently fails.
	gotCB, err := fakeGardenerClient.Resource(kgardener.CredentialsBindingResource).Namespace(gardenerNS).Get(context.Background(), cbName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "true", gotCB.GetLabels()["dirty"])
}

func newInjectBTPCBHelper(name, namespace string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetLabels(labels)
	u.SetGroupVersionKind(kgardener.CredentialsBindingGVK)
	return u
}

func strPtr(s string) *string { return &s }
