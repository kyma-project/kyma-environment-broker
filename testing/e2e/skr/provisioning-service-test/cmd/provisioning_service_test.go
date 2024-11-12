package cmd

import (
	"context"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/testing/e2e/skr/provisioning-service-test/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProvisioningService(t *testing.T) {
	suite := NewProvisioningSuite(t)

	suite.logger.Info("Fetching remaining environments")
	environments, err := suite.provisioningClient.GetEnvironments()
	require.NoError(t, err)
	suite.logger.Info("Environments fetched successfully", "Number of environments", len(environments.Environments))

	for _, environment := range environments.Environments {
		if environment.EnvironmentType == internal.KYMA {
			suite.logger.Info("Deleting remaining Kyma environment", "environmentID", environment.ID)
			_, err = suite.provisioningClient.DeleteEnvironment(environment.ID)
			require.NoError(t, err)

			err = suite.provisioningClient.AwaitEnvironmentDeleted(environment.ID)
			require.NoError(t, err)
			suite.logger.Info("Remaining Kyma environment deleted successfully", "environmentID", environment.ID)
		}
	}

	suite.logger.Info("Creating a new environment")
	environment, err := suite.provisioningClient.CreateEnvironment()
	require.NoError(t, err)

	err = suite.provisioningClient.AwaitEnvironmentCreated(environment.ID)
	assert.NoError(t, err)
	suite.logger.Info("Environment created successfully", "environmentID", environment.ID)

	suite.logger.Info("Creating a new binding")
	createdBinding, err := suite.provisioningClient.CreateBinding(environment.ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdBinding.Credentials.Kubeconfig)

	if len(createdBinding.Credentials.Kubeconfig) != 0 {
		suite.logger.Info("Creating a new K8s client set")
		clientset, err := suite.K8sClientSetForKubeconfig(createdBinding.Credentials.Kubeconfig)
		assert.NoError(t, err)

		suite.logger.Info("Fetching a secret", "Secret namespace", secretNamespace, "Secret name", secretName)
		_, err = clientset.CoreV1().Secrets(secretNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		assert.NoError(t, err)

		suite.logger.Info("Fetching a binding", "Binding ID", createdBinding.ID)
		fetchedBinding, err := suite.provisioningClient.GetBinding(environment.ID, createdBinding.ID)
		assert.NoError(t, err)
		assert.Equal(t, createdBinding.Credentials.Kubeconfig, fetchedBinding.Credentials.Kubeconfig)

		suite.logger.Info("Deleting a binding", "Binding ID", createdBinding.ID)
		err = suite.provisioningClient.DeleteBinding(environment.ID, createdBinding.ID)
		assert.NoError(t, err)

		suite.logger.Info("Trying to fetch a secret using invalidated kubeconfig", "Secret namespace", secretNamespace, "Secret name", secretName)
		_, err = clientset.CoreV1().Secrets(secretNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		assert.Error(t, err)

		suite.logger.Info("Trying to fetch a deleted binding", "Binding ID", createdBinding.ID)
		_, err = suite.provisioningClient.GetBinding(environment.ID, createdBinding.ID)
		assert.EqualError(t, err, "unexpected status code 404: body is empty")
	}

	suite.logger.Info("Deleting the environment", "environmentID", environment.ID)
	_, err = suite.provisioningClient.DeleteEnvironment(environment.ID)
	require.NoError(t, err)

	err = suite.provisioningClient.AwaitEnvironmentDeleted(environment.ID)
	assert.NoError(t, err)
	suite.logger.Info("Environment deleted successfully", "environmentID", environment.ID)
}
