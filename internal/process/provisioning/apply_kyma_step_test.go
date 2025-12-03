package provisioning

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

/*
Running tests with real K8S cluster instead of fake client.

k3d cluster create

kubectl create ns kyma-system

kubectl apply -f https://raw.githubusercontent.com/kyma-project/lifecycle-manager/main/operator/config/crd/bases/operator.kyma-project.io_kymas.yaml

k3d kubeconfig get --all > kubeconfig.yaml

export KUBECONFIG=kubeconfig.yaml

USE_KUBECONFIG=true go test -run TestCreatingKymaResource

kubectl get kymas -o yaml -n kyma-system
*/

func TestCreatingKymaResource(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.ProviderValues.ProviderType = "azure"
	*operation.ProvisioningParameters.ErsContext.LicenseType = "CUSTOMER"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	expectedLabels := map[string]string{
		"kyma-project.io/instance-id":         "inst-id",
		"kyma-project.io/runtime-id":          "runtime-inst-id",
		"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
		"kyma-project.io/subaccount-id":       "SA-op-id",
		"kyma-project.io/shoot-name":          "Shoot-inst-id",
		"kyma-project.io/platform-region":     "westeurope",
		"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
		"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		"kyma-project.io/broker-plan-name":    "azure",
		"operator.kyma-project.io/managed-by": "lifecycle-manager",
		"kyma-project.io/provider":            "Azure"}

	assertLabelsExistsForExternalKymaResource(t, expectedLabels, aList.Items[0])

	_, _, err = svc.Run(operation, fixLogger())
	require.NoError(t, err)
}

func TestCreatingKymaResourceWithCloudProviderInOperation(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.CloudProvider = "Azure"
	operation.ProviderValues.ProviderType = "azure"
	*operation.ProvisioningParameters.ErsContext.LicenseType = "CUSTOMER"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	expectedLabels := map[string]string{
		"kyma-project.io/instance-id":         "inst-id",
		"kyma-project.io/runtime-id":          "runtime-inst-id",
		"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
		"kyma-project.io/subaccount-id":       "SA-op-id",
		"kyma-project.io/shoot-name":          "Shoot-inst-id",
		"kyma-project.io/platform-region":     "westeurope",
		"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
		"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		"kyma-project.io/broker-plan-name":    "azure",
		"operator.kyma-project.io/managed-by": "lifecycle-manager",
		"kyma-project.io/provider":            "Azure"}

	assertLabelsExistsForExternalKymaResource(t, expectedLabels, aList.Items[0])

	_, _, err = svc.Run(operation, fixLogger())
	require.NoError(t, err)
}

func TestCreatingInternalKymaResource(t *testing.T) {
	t.Run("Without compass runtime ID", func(t *testing.T) {
		// given
		operation, cli := fixOperationForApplyKymaResource(t)
		operation.ProviderValues.ProviderType = "azure"
		storage := storage.NewMemoryStorage()
		err := storage.Operations().InsertOperation(operation)
		require.NoError(t, err)
		svc := NewApplyKymaStep(storage.Operations(), cli)

		// when
		_, backoff, err := svc.Run(operation, fixLogger())

		// then
		require.NoError(t, err)
		require.Zero(t, backoff)
		aList := unstructured.UnstructuredList{}
		aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

		err = cli.List(context.Background(), &aList)
		require.NoError(t, err)
		assert.Equal(t, 1, len(aList.Items))
		expectedLabels := map[string]string{
			"kyma-project.io/instance-id":         "inst-id",
			"kyma-project.io/runtime-id":          "runtime-inst-id",
			"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
			"kyma-project.io/subaccount-id":       "SA-op-id",
			"kyma-project.io/shoot-name":          "Shoot-inst-id",
			"kyma-project.io/platform-region":     "westeurope",
			"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
			"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
			"kyma-project.io/broker-plan-name":    "azure",
			"operator.kyma-project.io/managed-by": "lifecycle-manager",
			"kyma-project.io/provider":            "Azure"}
		assertLabelsExistsForInternalKymaResource(t, expectedLabels, aList.Items[0])
		_, _, err = svc.Run(operation, fixLogger())
		require.NoError(t, err)
	})
}

func TestCreatingKymaResource_UseNamespaceFromTimeOfCreationNotTemplate(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.KymaResourceNamespace = "namespace-in-time-of-creation"
	*operation.ProvisioningParameters.ErsContext.LicenseType = "CUSTOMER"
	operation.ProviderValues.ProviderType = "azure"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	expectedLabels := map[string]string{
		"kyma-project.io/instance-id":         "inst-id",
		"kyma-project.io/runtime-id":          "runtime-inst-id",
		"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
		"kyma-project.io/subaccount-id":       "SA-op-id",
		"kyma-project.io/shoot-name":          "Shoot-inst-id",
		"kyma-project.io/platform-region":     "westeurope",
		"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
		"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		"kyma-project.io/broker-plan-name":    "azure",
		"operator.kyma-project.io/managed-by": "lifecycle-manager",
		"kyma-project.io/provider":            "Azure"}
	assertLabelsExistsForExternalKymaResource(t, expectedLabels, aList.Items[0])

	_, _, err = svc.Run(operation, fixLogger())
	require.NoError(t, err)
	assert.Equal(t, "namespace-in-time-of-creation", operation.KymaResourceNamespace)
}

func TestCreatingInternalKymaResource_UseNamespaceFromTimeOfCreationNotTemplate(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.KymaResourceNamespace = "namespace-in-time-of-creation"
	operation.ProviderValues.ProviderType = "azure"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	expectedLabels := map[string]string{
		"kyma-project.io/instance-id":         "inst-id",
		"kyma-project.io/runtime-id":          "runtime-inst-id",
		"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
		"kyma-project.io/subaccount-id":       "SA-op-id",
		"kyma-project.io/shoot-name":          "Shoot-inst-id",
		"kyma-project.io/platform-region":     "westeurope",
		"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
		"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		"kyma-project.io/broker-plan-name":    "azure",
		"operator.kyma-project.io/managed-by": "lifecycle-manager",
		"kyma-project.io/provider":            "Azure"}
	assertLabelsExistsForInternalKymaResource(t, expectedLabels, aList.Items[0])

	_, _, err = svc.Run(operation, fixLogger())
	require.NoError(t, err)
	assert.Equal(t, "namespace-in-time-of-creation", operation.KymaResourceNamespace)
}

func TestUpdatinglKymaResourceIfExists(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.ProviderValues.ProviderType = "azure"
	*operation.ProvisioningParameters.ErsContext.LicenseType = "CUSTOMER"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)
	err = cli.Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operator.kyma-project.io/v1beta2",
		"kind":       "Kyma",
		"metadata": map[string]interface{}{
			"name":      operation.KymaResourceName,
			"namespace": "kyma-system",
		},
		"spec": map[string]interface{}{
			"channel": "fast",
		},
	}})
	require.NoError(t, err)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	expectedLabels := map[string]string{
		"kyma-project.io/instance-id":         "inst-id",
		"kyma-project.io/runtime-id":          "runtime-inst-id",
		"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
		"kyma-project.io/subaccount-id":       "SA-op-id",
		"kyma-project.io/shoot-name":          "Shoot-inst-id",
		"kyma-project.io/platform-region":     "westeurope",
		"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
		"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		"kyma-project.io/broker-plan-name":    "azure",
		"operator.kyma-project.io/managed-by": "lifecycle-manager",
		"kyma-project.io/provider":            "Azure"}

	assertLabelsExistsForExternalKymaResource(t, expectedLabels, aList.Items[0])
}

func TestUpdatinInternalKymaResourceIfExists(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.ProviderValues.ProviderType = "azure"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)
	err = cli.Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operator.kyma-project.io/v1beta2",
		"kind":       "Kyma",
		"metadata": map[string]interface{}{
			"name":      operation.KymaResourceName,
			"namespace": "kyma-system",
		},
		"spec": map[string]interface{}{
			"channel": "fast",
		},
	}})
	require.NoError(t, err)

	// when
	_, backoff, err := svc.Run(operation, fixLogger())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	expectedLabels := map[string]string{
		"kyma-project.io/instance-id":         "inst-id",
		"kyma-project.io/runtime-id":          "runtime-inst-id",
		"kyma-project.io/global-account-id":   "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4",
		"kyma-project.io/subaccount-id":       "SA-op-id",
		"kyma-project.io/shoot-name":          "Shoot-inst-id",
		"kyma-project.io/platform-region":     "westeurope",
		"operator.kyma-project.io/kyma-name":  "runtime-inst-id",
		"kyma-project.io/broker-plan-id":      "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		"kyma-project.io/broker-plan-name":    "azure",
		"operator.kyma-project.io/managed-by": "lifecycle-manager",
		"kyma-project.io/provider":            "Azure"}

	assertLabelsExistsForInternalKymaResource(t, expectedLabels, aList.Items[0])
}

func assertLabelsExists(t *testing.T, expectedLabels map[string]string, obj unstructured.Unstructured) {
	keys := make([]string, 0, len(obj.GetLabels()))
	for k := range obj.GetLabels() {
		keys = append(keys, k)
	}

	if !assert.Subset(t, obj.GetLabels(), expectedLabels) {
		t.Logf("Expected labels: %v", expectedLabels)
		t.Logf("Actual labels:   %v", obj.GetLabels())
	}
}

func assertLabelsExistsForInternalKymaResource(t *testing.T, expectedLabels map[string]string, obj unstructured.Unstructured) {
	assert.Contains(t, obj.GetLabels(), "operator.kyma-project.io/internal")
	assertLabelsExists(t, expectedLabels, obj)
}

func assertLabelsExistsForExternalKymaResource(t *testing.T, expectedLabels map[string]string, obj unstructured.Unstructured) {
	assert.NotContains(t, obj.GetLabels(), "operator.kyma-project.io/internal")
	assertLabelsExists(t, expectedLabels, obj)
}

func fixOperationForApplyKymaResource(t *testing.T) (internal.Operation, client.Client) {
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.ProviderValues = &internal.ProviderValues{
		ProviderType: "aws",
	}
	operation.KymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
    name: my-kyma
    namespace: kyma-system
spec:
    sync:
        strategy: secret
    channel: fast
    modules: []
`
	var cli client.Client
	if len(os.Getenv("KUBECONFIG")) > 0 && strings.ToLower(os.Getenv("USE_KUBECONFIG")) == "true" {
		config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			t.Fatal(err.Error())
		}
		// controller-runtime lib
		scheme.Scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "operator.kyma-project.io",
			Version: "v1beta2",
			Kind:    "kyma",
		}, &unstructured.Unstructured{})

		cli, err = client.New(config, client.Options{})
		if err != nil {
			t.Fatal(err.Error())
		}
	} else {
		cli = fake.NewClientBuilder().Build()
	}

	return operation, cli
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
