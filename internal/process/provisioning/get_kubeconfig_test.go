package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	kubeconfigContentsFromParameters = "apiVersion: v1"
	kubeconfigFromRuntime            = "kubeconfig-content"
	kubeconfigFromPreviousOperation  = "kubeconfig-already-set"
)

func TestGetKubeconfigStep(t *testing.T) {

	kimConfig := broker.KimConfig{
		Enabled: false,
	}

	t.Run("should create k8s client using kubeconfig from RuntimeStatus", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()
		provisionerClient := provisioner.NewFakeClient()

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)

		step := NewGetKubeconfigStep(st.Operations(), provisionerClient, kimConfig)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = ""
		err = st.Operations().InsertOperation(operation)
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		// when
		processedOperation, d, err := step.Run(operation, fixLogger())

		// then
		require.NoError(t, err)
		assert.Zero(t, d)
		assert.Equal(t, kubeconfigFromRuntime, processedOperation.Kubeconfig)
		assert.NotEmpty(t, processedOperation.Kubeconfig)
	})
	t.Run("should create k8s client for own_cluster plan using kubeconfig from provisioning parameters", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)

		step := NewGetKubeconfigStep(st.Operations(), nil, kimConfig)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = ""
		operation.ProvisioningParameters.Parameters.Kubeconfig = kubeconfigContentsFromParameters
		operation.ProvisioningParameters.PlanID = broker.OwnClusterPlanID
		err = st.Operations().InsertOperation(operation)
		require.NoError(t, err)

		// when
		processedOperation, d, err := step.Run(operation, fixLogger())

		// then
		require.NoError(t, err)
		assert.Zero(t, d)
		assert.Equal(t, kubeconfigContentsFromParameters, processedOperation.Kubeconfig)
	})
	t.Run("should create k8s client using kubeconfig already set in operation", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)

		step := NewGetKubeconfigStep(st.Operations(), nil, kimConfig)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = kubeconfigFromPreviousOperation
		operation.ProvisioningParameters.Parameters.Kubeconfig = ""
		err = st.Operations().InsertOperation(operation)
		require.NoError(t, err)

		// when
		processedOperation, d, err := step.Run(operation, fixLogger())

		// then
		require.NoError(t, err)
		assert.Zero(t, d)
		assert.Equal(t, kubeconfigFromPreviousOperation, processedOperation.Kubeconfig)
	})
	t.Run("should fail with error if there is neither kubeconfig nor runtimeID and this is not own_cluster plan", func(t *testing.T) {
		// given
		st := storage.NewMemoryStorage()
		provisionerClient := provisioner.NewFakeClient()

		scheme := internal.NewSchemeForTests(t)
		err := apiextensionsv1.AddToScheme(scheme)

		step := NewGetKubeconfigStep(st.Operations(), provisionerClient, kimConfig)
		operation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		operation.Kubeconfig = ""
		operation.RuntimeID = ""
		err = st.Operations().InsertOperation(operation)
		require.NoError(t, err)

		// when
		_, _, err = step.Run(operation, fixLogger())

		// then
		require.ErrorContains(t, err, "Runtime ID is empty")
	})
}
