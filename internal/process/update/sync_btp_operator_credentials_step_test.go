package update

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	btpmanagercredentials "github.com/kyma-project/kyma-environment-broker/internal/btpmanager/credentials"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/pivotal-cf/brokerapi/v12/domain"

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

func TestSyncBTPOperatorCredentialsStep_SecretAbsent(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// when
	result, backoff, err := step.Run(operation, fixUpdateLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, backoff)
	assertUpdateSecretContent(t, k8sClient, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.InstanceDetails.ServiceManagerClusterID)
	assert.Equal(t, "BTP operator secret updated with new credentials", result.Description)
}

func TestSyncBTPOperatorCredentialsStep_SecretMatchingNoWrite(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

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
	result, backoff, err := step.Run(operation, fixUpdateLogger())

	// then — no write, ResourceVersion unchanged, description set
	require.NoError(t, err)
	assert.Zero(t, backoff)
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.Equal(t, rvBefore, secret.ResourceVersion)
	assert.Equal(t, "BTP operator secret already up to date — no write needed", result.Description)
}

func TestSyncBTPOperatorCredentialsStep_SecretUpdatedOnRotation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// first run creates the secret
	_, _, err := step.Run(operation, fixUpdateLogger())
	require.NoError(t, err)

	// when — credentials are rotated
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = "rotated-secret"
	result, backoff, err := step.Run(operation, fixUpdateLogger())

	// then — secret updated with new value, description set
	require.NoError(t, err)
	assert.Zero(t, backoff)
	assertUpdateSecretContent(t, k8sClient, operation.ProvisioningParameters.ErsContext.SMOperatorCredentials, operation.InstanceDetails.ServiceManagerClusterID)
	assert.Equal(t, "BTP operator secret updated with new credentials", result.Description)
}

func TestSyncBTPOperatorCredentialsStep_EmptyClusterIDGeneratesAndWrites(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.InstanceDetails.ServiceManagerClusterID = ""
	// must persist the operation so UpdateOperation can find it
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

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

func TestSyncBTPOperatorCredentialsStep_SKRClientUnavailableRetries(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	operation := fixUpdateOperationWithClusterIDAndCredentials()
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(nil))

	// when
	_, backoff, _ := step.Run(operation, fixUpdateLogger())

	// then — retries on unavailable SKR client
	assert.NotZero(t, backoff)
}

func TestSyncBTPOperatorCredentialsStep_SecretUpdatedOnPartialCredentials(t *testing.T) {
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

	// In the real pipeline, dto.go merges partial creds with existing DB values before the step runs.
	// The operation therefore carries fully-merged credentials — only clientsecret is new.
	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientID:          "original-client-id",
		ClientSecret:      "rotated-secret",
		ServiceManagerURL: "https://sm.url",
		URL:               "https://token.url",
		XSAppName:         "sample-app-name",
	}
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

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
}

func TestSyncBTPOperatorCredentialsStep_SkipReconciliationLabel_UserManagedSecret(t *testing.T) {
	// given — secret has skip-reconciliation label AND stable fields (sm_url, tokenurl) differ
	// from the incoming credentials — user intentionally manages this secret
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	existingSecret := &apicorev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      btpmanagercredentials.BtpManagerSecretName,
			Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
			Labels: map[string]string{
				btpmanagercredentials.SkipReconciliationLabel: "true",
			},
			ResourceVersion: "1",
		},
		Data: map[string][]byte{
			"clientid":     []byte("different-client-id"),
			"clientsecret": []byte("user-managed-secret"),
			"sm_url":       []byte("https://user-managed-sm.url"),    // differs from incoming creds
			"tokenurl":     []byte("https://user-managed-token.url"), // differs from incoming creds
			"cluster_id":   []byte("cluster-id"),
		},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = "rotated-secret"
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// capture log output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	// when
	result, backoff, err := step.Run(operation, logger)

	// then — step skips entirely, secret not touched
	require.NoError(t, err)
	assert.Zero(t, backoff)
	secret := &apicorev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.Equal(t, "1", secret.ResourceVersion, "secret must not be written when user manages it")
	assert.Equal(t, "true", secret.Labels[btpmanagercredentials.SkipReconciliationLabel], "skip-reconciliation label must be preserved")

	// then — description records the user-managed skip outcome
	assert.Contains(t, result.Description, "user manages this secret")

	// then — log explains WHY the skip happened (visible in Plutono by runtimeID)
	logOutput := logBuf.String()
	assert.True(t, strings.Contains(logOutput, "skip-reconciliation label"),
		"expected skip-reconciliation label message in logs, got: %s", logOutput)
	assert.True(t, strings.Contains(logOutput, operation.RuntimeID),
		"expected runtimeID in skip log, got: %s", logOutput)
}

func TestSyncBTPOperatorCredentialsStep_SkipReconciliationLabel_RotationOnly(t *testing.T) {
	// given — secret has skip-reconciliation label but stable fields (sm_url, tokenurl) match
	// the incoming credentials — the label was set only to protect the secret during rotation
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials.ClientSecret = "rotated-secret"

	// secret has the same clientid/sm_url/tokenurl as the incoming creds, but old clientsecret
	existingSecret := &apicorev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      btpmanagercredentials.BtpManagerSecretName,
			Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
			Labels: map[string]string{
				btpmanagercredentials.SkipReconciliationLabel: "true",
			},
			ResourceVersion: "1",
		},
		Data: map[string][]byte{
			"clientid":     []byte(creds.ClientID),
			"clientsecret": []byte("old-secret"),
			"sm_url":       []byte(creds.ServiceManagerURL),
			"tokenurl":     []byte(creds.URL),
			"cluster_id":   []byte(operation.InstanceDetails.ServiceManagerClusterID),
		},
	}
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// capture log output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	// when
	result, backoff, err := step.Run(operation, logger)

	// then — label removed and new clientsecret written
	require.NoError(t, err)
	assert.Zero(t, backoff)
	secret := &apicorev1.Secret{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: btpmanagercredentials.BtpManagerSecretNamespace,
		Name:      btpmanagercredentials.BtpManagerSecretName,
	}, secret))
	assert.Equal(t, []byte("rotated-secret"), secret.Data["clientsecret"], "new clientsecret must be written")
	assert.NotEqual(t, "true", secret.Labels[btpmanagercredentials.SkipReconciliationLabel], "skip-reconciliation label must be removed")

	// then — description records label-removal + credential sync
	assert.Contains(t, result.Description, "skip-reconciliation label removed")

	// then — log records that label was removed
	logOutput := logBuf.String()
	assert.True(t, strings.Contains(logOutput, "removing label"),
		"expected label-removal message in logs, got: %s", logOutput)
}

func TestSyncBTPOperatorCredentialsStep_IncompleteCredsFailsOperation(t *testing.T) {
	// given — credentials missing ClientID (would fail PrepareSecret)
	memoryStorage := storage.NewMemoryStorage()
	scheme := internal.NewSchemeForTests(t)
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	operation := fixUpdateOperationWithClusterIDAndCredentials()
	operation.ProvisioningParameters.ErsContext.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
		ClientID:          "",
		ClientSecret:      "secret",
		ServiceManagerURL: "https://sm.url",
		URL:               "https://token.url",
	}
	require.NoError(t, memoryStorage.Operations().InsertOperation(operation))
	step := NewSyncBTPOperatorCredentialsStep(memoryStorage.Operations(), kubeconfig.NewFakeK8sClientProvider(k8sClient))

	// when
	processedOperation, backoff, _ := step.Run(operation, fixUpdateLogger())

	// then — operation fails permanently, not retried
	assert.Zero(t, backoff)
	assert.Equal(t, domain.Failed, processedOperation.State)
}

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

func assertUpdateSecretContent(t *testing.T, k8sClient client.Client, creds *internal.ServiceManagerOperatorCredentials, clusterID string) {
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

func fixUpdateLogger() *slog.Logger {
	return slog.Default()
}
