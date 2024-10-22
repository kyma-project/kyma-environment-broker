package provisioningservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvisioningService(t *testing.T) {
	suite := NewProvisioningSuite(t)

	suite.logger.Info("Creating a new environment")
	environment, err := suite.provisioningClient.CreateEnvironment()
	require.NoError(t, err)

	err = suite.provisioningClient.AwaitEnvironmentCreated(environment.ID)
	require.NoError(t, err)
	suite.logger.Info("Environment created successfully", "environmentID", environment.ID)

	suite.logger.Info("Creating a new binding")
	createdBinding, err := suite.provisioningClient.CreateBinding(environment.ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdBinding.Credentials.Kubeconfig)

	suite.logger.Info("Retrieving a binding", "Binding ID", createdBinding.ID)
	retrievedBinding, err := suite.provisioningClient.GetBinding(environment.ID, createdBinding.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdBinding.Credentials.Kubeconfig, retrievedBinding.Credentials.Kubeconfig)

	suite.logger.Info("Deleting a binding", "Binding ID", createdBinding.ID)
	err = suite.provisioningClient.DeleteBinding(environment.ID, createdBinding.ID)
	assert.NoError(t, err)

	suite.logger.Info("Retrieving a binding", "Binding ID", createdBinding.ID)
	_, err = suite.provisioningClient.GetBinding(environment.ID, createdBinding.ID)
	assert.EqualError(t, err, "unexpected status code: 400")

	suite.logger.Info("Deleting the environment", "environmentID", environment.ID)
	_, err = suite.provisioningClient.DeleteEnvironment(environment.ID)
	require.NoError(t, err)

	err = suite.provisioningClient.AwaitEnvironmentDeleted(environment.ID)
	require.NoError(t, err)
	suite.logger.Info("Environment deleted successfully", "environmentID", environment.ID)
}
