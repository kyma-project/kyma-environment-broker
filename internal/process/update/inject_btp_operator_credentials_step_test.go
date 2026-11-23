package update

import (
	"context"
	"log/slog"
	"testing"

	btpmanagercredentials "github.com/kyma-project/kyma-environment-broker/internal/btpmanager/credentials"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	apicorev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateInjectBTPOperatorCredentialsStep_SecretAbsent(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// when
	_, backoff, err := step.Run(operation, fixUpdateLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, backoff)
	assertUpdateSecretMatchesCredentials(t, k8sClient, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.InstanceDetails.ServiceManagerClusterID)
	assertUpdateNamespacePresent(t, k8sClient)
}

func TestUpdateInjectBTPOperatorCredentialsStep_SecretMatchingNoWrite(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// first run creates the secret
	_, _, err := step.Run(operation, fixUpdateLogger())
	require.NoError(t, err)

	// capture ResourceVersion after first write
	secret := &apicorev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	rvBefore := secret.ResourceVersion

	// when — run again with same credentials
	_, backoff, err := step.Run(operation, fixUpdateLogger())

	// then — no write, ResourceVersion unchanged
	require.NoError(t, err)
	assert.Zero(t, backoff)
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.Equal(t, rvBefore, secret.ResourceVersion)
}

func TestUpdateInjectBTPOperatorCredentialsStep_SecretUpdatedOnRotation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// first run creates the secret
	_, _, err := step.Run(operation, fixUpdateLogger())
	require.NoError(t, err)

	// when — credentials are rotated
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = "rotated-secret"
	_, backoff, err := step.Run(operation, fixUpdateLogger())

	// then — secret updated with new value
	require.NoError(t, err)
	assert.Zero(t, backoff)
	assertUpdateSecretMatchesCredentials(t, k8sClient, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.InstanceDetails.ServiceManagerClusterID)
}

func TestUpdateInjectBTPOperatorCredentialsStep_EmptyRuntimeIDRetries(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.RuntimeID = ""
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// when
	processedOperation, backoff, _ := step.Run(operation, fixUpdateLogger())

	// then — retries, does NOT fail the operation permanently
	assert.NotZero(t, backoff)
	assert.NotEqual(t, "Failed", string(processedOperation.State))
}

func TestUpdateInjectBTPOperatorCredentialsStep_EmptyClusterIDGeneratesAndWrites(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.InstanceDetails.ServiceManagerClusterID = ""
	// must persist the operation so UpdateOperation can find it
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// when
	_, backoff, err := step.Run(operation, fixUpdateLogger())

	// then — cluster ID generated, secret written
	require.NoError(t, err)
	assert.Zero(t, backoff)
	secret := &apicorev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.NotEmpty(t, secret.Data["cluster_id"])
}

func TestUpdateInjectBTPOperatorCredentialsStep_SKRClientUnavailableRetries(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	operation := fixUpdateOperationWithClusterIDAndCredentials()
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(nil))

	// when
	_, backoff, _ := step.Run(operation, fixUpdateLogger())

	// then — retries on unavailable SKR client
	assert.NotZero(t, backoff)
}

func TestUpdateInjectBTPOperatorCredentialsStep_SecretUpdatedOnPartialCredentials(t *testing.T) {
	// given — secret already exists with full credentials
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	existingSecret := &apicorev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      btpmanagercredentials.BtpManagerSecretName,
			Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		},
		Data: map[string][]byte{
			"clientid":     []byte("original-client-id"),
			"clientsecret": []byte("original-secret"),
			"sm_url":       []byte("https://sm.url"),
			"tokenurl":     []byte("https://token.url"),
			"cluster_id":   []byte("cluster-id"),
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()

	// operation carries only the rotated clientsecret — all other fields are empty strings
	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientSecret: "rotated-secret",
	}
	step := NewInjectBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// when
	_, backoff, err := step.Run(operation, fixUpdateLogger())

	// then — only clientsecret updated, other keys preserved from existing secret
	require.NoError(t, err)
	assert.Zero(t, backoff)
	secret := &apicorev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.Equal(t, []byte("rotated-secret"), secret.Data["clientsecret"])
	assert.Equal(t, []byte("original-client-id"), secret.Data["clientid"])
	assert.Equal(t, []byte("https://sm.url"), secret.Data["sm_url"])
	assert.Equal(t, []byte("https://token.url"), secret.Data["tokenurl"])
	assert.Equal(t, []byte("cluster-id"), secret.Data["cluster_id"])
}

// helpers

func fixUpdateOperationWithClusterIDAndCredentials() internal.Operation {
	operation := fixture.FixUpdatingOperation("operation-id", "inst-id")
	operation.RuntimeID = "runtime-id"
	operation.InstanceDetails.ServiceManagerClusterID = "cluster-id"
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientID:          "sample-client-id",
		ClientSecret:      "sample-client-secret",
		ServiceManagerURL: "www.service.manager.url.com",
		URL:               "www.sample.url.com",
		XSAppName:         "sample-app-name",
	}
	return operation
}

func assertUpdateSecretMatchesCredentials(t *testing.T, k8sClient client.Client, creds *internal.ServiceManagerOperatorCredentials, clusterID string) {
	t.Helper()
	secret := &apicorev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.Equal(t, []byte(creds.ClientID), secret.Data["clientid"])
	assert.Equal(t, []byte(creds.ClientSecret), secret.Data["clientsecret"])
	assert.Equal(t, []byte(creds.ServiceManagerURL), secret.Data["sm_url"])
	assert.Equal(t, []byte(creds.URL), secret.Data["tokenurl"])
	assert.Equal(t, []byte(clusterID), secret.Data["cluster_id"])
	assert.Equal(t, btpmanagercredentials.BtpManagerLabels, secret.Labels)
	assert.Equal(t, btpmanagercredentials.BtpManagerAnnotations, secret.Annotations)
}

func assertUpdateNamespacePresent(t *testing.T, k8sClient client.Client) {
	t.Helper()
	ns := &apicorev1.Namespace{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Name: btpmanagercredentials.BtpManagerSecretNamespace,
	}, ns))
}

func fixUpdateLogger() *slog.Logger {
	return slog.Default()
}
