package provisioning

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
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
	*operation.ProvisioningParameters.ErsContext.LicenseType = "CUSTOMER"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExistsForExternalKymaResource(t, aList.Items[0])

	_, _, err = svc.Run(operation, logrus.New())
	require.NoError(t, err)
}

func TestCreatingInternalKymaResource(t *testing.T) {
	t.Run("With compass runtime ID", func(t *testing.T) {
		// given
		operation, cli := fixOperationForApplyKymaResource(t)
		storage := storage.NewMemoryStorage()
		err := storage.Operations().InsertOperation(operation)
		require.NoError(t, err)
		svc := NewApplyKymaStep(storage.Operations(), cli)

		// when
		_, backoff, err := svc.Run(operation, logrus.New())

		// then
		require.NoError(t, err)
		require.Zero(t, backoff)
		aList := unstructured.UnstructuredList{}
		aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

		err = cli.List(context.Background(), &aList)
		require.NoError(t, err)
		assert.Equal(t, 1, len(aList.Items))
		assertLabelsExistsForInternalKymaResource(t, aList.Items[0])

		assertCompassRuntimeIdAnnotationExists(t, aList.Items[0])
		_, _, err = svc.Run(operation, logrus.New())
		require.NoError(t, err)
	})

	t.Run("Without compass runtime ID", func(t *testing.T) {
		// given
		operation, cli := fixOperationForApplyKymaResource(t)
		operation.SetCompassRuntimeIdNotRegisteredByProvisioner()
		storage := storage.NewMemoryStorage()
		err := storage.Operations().InsertOperation(operation)
		require.NoError(t, err)
		svc := NewApplyKymaStep(storage.Operations(), cli)

		// when
		_, backoff, err := svc.Run(operation, logrus.New())

		// then
		require.NoError(t, err)
		require.Zero(t, backoff)
		aList := unstructured.UnstructuredList{}
		aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

		err = cli.List(context.Background(), &aList)
		require.NoError(t, err)
		assert.Equal(t, 1, len(aList.Items))
		assertLabelsExistsForInternalKymaResource(t, aList.Items[0])

		assertCompassRuntimeIdAnnotationNotExists(t, aList.Items[0])
		_, _, err = svc.Run(operation, logrus.New())
		require.NoError(t, err)
	})
}

func TestCreatingKymaResource_UseNamespaceFromTimeOfCreationNotTemplate(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.KymaResourceNamespace = "namespace-in-time-of-creation"
	*operation.ProvisioningParameters.ErsContext.LicenseType = "CUSTOMER"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExistsForExternalKymaResource(t, aList.Items[0])

	_, _, err = svc.Run(operation, logrus.New())
	require.NoError(t, err)
	assert.Equal(t, "namespace-in-time-of-creation", operation.KymaResourceNamespace)
}

func TestCreatingInternalKymaResource_UseNamespaceFromTimeOfCreationNotTemplate(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
	operation.KymaResourceNamespace = "namespace-in-time-of-creation"
	storage := storage.NewMemoryStorage()
	err := storage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewApplyKymaStep(storage.Operations(), cli)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExistsForInternalKymaResource(t, aList.Items[0])

	_, _, err = svc.Run(operation, logrus.New())
	require.NoError(t, err)
	assert.Equal(t, "namespace-in-time-of-creation", operation.KymaResourceNamespace)
}

func TestUpdatinglKymaResourceIfExists(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
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
			"channel": "stable",
		},
	}})
	require.NoError(t, err)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExistsForExternalKymaResource(t, aList.Items[0])
}

func TestUpdatinInternalKymaResourceIfExists(t *testing.T) {
	// given
	operation, cli := fixOperationForApplyKymaResource(t)
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
			"channel": "stable",
		},
	}})
	require.NoError(t, err)

	// when
	_, backoff, err := svc.Run(operation, logrus.New())

	// then
	require.NoError(t, err)
	require.Zero(t, backoff)
	aList := unstructured.UnstructuredList{}
	aList.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "KymaList"})

	err = cli.List(context.Background(), &aList)
	require.NoError(t, err)
	assert.Equal(t, 1, len(aList.Items))
	assertLabelsExistsForInternalKymaResource(t, aList.Items[0])
}

func assertLabelsExists(t *testing.T, obj unstructured.Unstructured) {
	keys := make([]string, 0, len(obj.GetLabels()))
	for k := range obj.GetLabels() {
		keys = append(keys, k)
	}

	assert.Subset(t, keys, []string{
		"kyma-project.io/instance-id",
		"kyma-project.io/runtime-id",
		"kyma-project.io/global-account-id",
		"kyma-project.io/subaccount-id",
		"kyma-project.io/shoot-name",
		"kyma-project.io/platform-region",
		"operator.kyma-project.io/kyma-name",
		"kyma-project.io/broker-plan-id",
		"kyma-project.io/broker-plan-name",
		"operator.kyma-project.io/managed-by",
		"kyma-project.io/provider"})
}

func assertLabelsExistsForInternalKymaResource(t *testing.T, obj unstructured.Unstructured) {
	assert.Contains(t, obj.GetLabels(), "operator.kyma-project.io/internal")
	assertLabelsExists(t, obj)
}

func assertCompassRuntimeIdAnnotationExists(t *testing.T, obj unstructured.Unstructured) {
	t.Helper()
	assert.Contains(t, obj.GetAnnotations(), "compass-runtime-id-for-migration")
}

func assertCompassRuntimeIdAnnotationNotExists(t *testing.T, obj unstructured.Unstructured) {
	t.Helper()
	assert.NotContains(t, obj.GetAnnotations(), "compass-runtime-id-for-migration")
}

func assertLabelsExistsForExternalKymaResource(t *testing.T, obj unstructured.Unstructured) {
	assert.NotContains(t, obj.GetLabels(), "operator.kyma-project.io/internal")
	assertLabelsExists(t, obj)
}

func fixOperationForApplyKymaResource(t *testing.T) (internal.Operation, client.Client) {
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	operation.KymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
    name: my-kyma
    namespace: kyma-system
spec:
    sync:
        strategy: secret
    channel: stable
    modules: []
`
	operation.InputCreator = fixture.FixInputCreator("Test")
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
		fmt.Println("using kubeconfig")
	} else {
		fmt.Println("using fake client")
		cli = fake.NewClientBuilder().Build()
	}

	return operation, cli
}
