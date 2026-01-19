package provisioning

import (
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCredentialsBindingStep(t *testing.T) {
	// given
	brokerStorage := storage.NewMemoryStorage()
	gardenerClient := fixture.CreateGardenerClientWithCredentialsBindings()
	rulesService := createRulesService(t)
	stepRetryTuple := internal.RetryTuple{
		Timeout:  2 * time.Second,
		Interval: 1 * time.Second,
	}
	immediateTimeout := internal.RetryTuple{
		Timeout:  -1 * time.Second,
		Interval: 1 * time.Second,
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	t.Run("should resolve secret name for aws hyperscaler and existing tenant", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-1"
			instanceID     = "instance-1"
			platformRegion = "cf-eu11"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = fixture.AWSTenantName
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, fixture.AWSEUAccessClaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSEUAccessClaimedSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should resolve secret name for azure hyperscaler and existing tenant", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-2"
			instanceID     = "instance-2"
			platformRegion = "cf-ch20"
			providerType   = "azure"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.Azure)))
		operation.ProvisioningParameters.PlanID = broker.AzurePlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = fixture.AzureTenantName
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, fixture.AzureEUAccessClaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AzureEUAccessClaimedSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should resolve unclaimed secret name for azure hyperscaler", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-3"
			instanceID     = "instance-3"
			platformRegion = "cf-ap21"
			providerType   = "azure"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.Azure)))
		operation.ProvisioningParameters.PlanID = broker.AzurePlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, fixture.AzureUnclaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AzureUnclaimedSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should resolve shared secret name for gcp hyperscaler", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-4"
			instanceID     = "instance-4"
			platformRegion = "cf-eu30"
			providerType   = "gcp"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.GCP)))
		operation.ProvisioningParameters.PlanID = broker.GCPPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, fixture.GCPEUAccessSharedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.GCPEUAccessSharedSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should resolve the least used shared secret name for aws hyperscaler and trial plan", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-5"
			instanceID     = "instance-5"
			platformRegion = "cf-eu10"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.TrialPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, fixture.AWSLeastUsedSharedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSLeastUsedSharedSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should return error on missing rule match for given provisioning attributes", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-6"
			instanceID     = "instance-6"
			platformRegion = "non-existent-region"
			providerType   = "openstack"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.SapConvergedCloud)))
		operation.ProvisioningParameters.PlanID = broker.SapConvergedCloudPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, immediateTimeout)

		// when
		_, backoff, err := step.Run(operation, log)

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.True(t, strings.Contains(err.Error(), "no matching rule for provisioning attributes"))

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Empty(t, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should return error on missing secret binding for given selector", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-7"
			instanceID     = "instance-7"
			platformRegion = "cf-ap11"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, immediateTimeout)

		// when
		_, backoff, err := step.Run(operation, log)

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.True(t, strings.Contains(err.Error(), "failed to find unassigned secret binding with selector"))

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Empty(t, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should fail operation when target secret name is empty", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-8"
			instanceID     = "instance-8"
			platformRegion = "cf-us30"
			providerType   = "gcp"
		)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.GCP)))
		operation.ProvisioningParameters.PlanID = broker.GCPPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, immediateTimeout)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.Error(t, err)
		assert.Zero(t, backoff)
		assert.ErrorContains(t, err, "failed to determine secret name")
		assert.Equal(t, domain.Failed, operation.State)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Empty(t, updatedInstance.SubscriptionSecretName)
	})
}
