package provisioningservice

import (
	"testing"

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

	suite.logger.Info("Deleting the environment", "environmentID", environment.ID)
	_, err = suite.provisioningClient.DeleteEnvironment(environment.ID)
	require.NoError(t, err)

	err = suite.provisioningClient.AwaitEnvironmentDeleted(environment.ID)
	require.NoError(t, err)
	suite.logger.Info("Environment deleted successfully", "environmentID", environment.ID)
}
