package provisioning

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/multiaccount"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func disabledMultiAccountConfig() *multiaccount.MultiAccountConfig {
	return &multiaccount.MultiAccountConfig{
		AllowedGlobalAccounts: []string{},
	}
}

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, disabledMultiAccountConfig())

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, disabledMultiAccountConfig())

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, disabledMultiAccountConfig())

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, disabledMultiAccountConfig())

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, disabledMultiAccountConfig())

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, immediateTimeout, disabledMultiAccountConfig())

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, immediateTimeout, disabledMultiAccountConfig())

		// when
		_, backoff, err := step.Run(operation, log)

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.True(t, strings.Contains(err.Error(), "failed to find unassigned credentials binding with selector"))

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

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, immediateTimeout, disabledMultiAccountConfig())

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

func TestGetLimitForProvider(t *testing.T) {
	t.Run("should return provider-specific limit when set", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			Limits: multiaccount.HyperscalerAccountLimits{
				Default:   100,
				AWS:       180,
				GCP:       135,
				Azure:     135,
				OpenStack: 100,
				AliCloud:  100,
			},
		}

		assert.Equal(t, 180, getLimitForProvider(config, "aws"))
		assert.Equal(t, 135, getLimitForProvider(config, "gcp"))
		assert.Equal(t, 135, getLimitForProvider(config, "azure"))
		assert.Equal(t, 100, getLimitForProvider(config, "openstack"))
		assert.Equal(t, 100, getLimitForProvider(config, "alicloud"))
	})

	t.Run("should return default limit for unknown provider", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			Limits: multiaccount.HyperscalerAccountLimits{
				Default:   250,
				AWS:       180,
				GCP:       135,
				Azure:     135,
				OpenStack: 100,
				AliCloud:  100,
			},
		}

		assert.Equal(t, 250, getLimitForProvider(config, "unknown-provider"))
		assert.Equal(t, 250, getLimitForProvider(config, "new-hyperscaler"))
	})

	t.Run("should return default limit when provider-specific limit is zero", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			Limits: multiaccount.HyperscalerAccountLimits{
				Default:   150,
				AWS:       0,
				GCP:       135,
				Azure:     0,
				OpenStack: 100,
			},
		}

		assert.Equal(t, 150, getLimitForProvider(config, "aws"))
		assert.Equal(t, 135, getLimitForProvider(config, "gcp"))
		assert.Equal(t, 150, getLimitForProvider(config, "azure"))
		assert.Equal(t, 100, getLimitForProvider(config, "openstack"))
		assert.Equal(t, 150, getLimitForProvider(config, "alicloud"))
	})

	t.Run("should return 999999 when default is zero", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			Limits: multiaccount.HyperscalerAccountLimits{
				Default: 0,
				AWS:     0,
				GCP:     0,
			},
		}

		assert.Equal(t, 999999, getLimitForProvider(config, "aws"))
		assert.Equal(t, 999999, getLimitForProvider(config, "gcp"))
		assert.Equal(t, 999999, getLimitForProvider(config, "azure"))
		assert.Equal(t, 999999, getLimitForProvider(config, "unknown"))
	})

	t.Run("should return 0 when config is nil", func(t *testing.T) {
		assert.Equal(t, 0, getLimitForProvider(nil, "aws"))
		assert.Equal(t, 0, getLimitForProvider(nil, "unknown"))
	})

	t.Run("should handle openstack provider name variants", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			Limits: multiaccount.HyperscalerAccountLimits{
				Default:   100,
				OpenStack: 75,
			},
		}

		// All these variants map to SapConvergedCloud → OpenStack limit
		assert.Equal(t, 75, getLimitForProvider(config, "openstack"))
		assert.Equal(t, 75, getLimitForProvider(config, "sapconvergedcloud"))
		assert.Equal(t, 75, getLimitForProvider(config, "sap-converged-cloud"))
	})

	t.Run("should use default when OpenStack limit is not set", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			Limits: multiaccount.HyperscalerAccountLimits{
				Default:   200,
				OpenStack: 0,
			},
		}

		assert.Equal(t, 200, getLimitForProvider(config, "openstack"))
		assert.Equal(t, 200, getLimitForProvider(config, "sapconvergedcloud"))
	})
}

func TestIsMultiAccountEnabled(t *testing.T) {
	t.Run("should return true when allowed global accounts is not empty", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-123"},
		}
		assert.True(t, isMultiAccountEnabled(config))
	})

	t.Run("should return true when allowed global accounts has wildcard", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"*"},
		}
		assert.True(t, isMultiAccountEnabled(config))
	})

	t.Run("should return true when allowed global accounts has multiple entries", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-1", "ga-2", "ga-3"},
		}
		assert.True(t, isMultiAccountEnabled(config))
	})

	t.Run("should return false when allowed global accounts is empty", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{},
		}
		assert.False(t, isMultiAccountEnabled(config))
	})

	t.Run("should return false when config is nil", func(t *testing.T) {
		assert.False(t, isMultiAccountEnabled(nil))
	})
}

func TestIsGlobalAccountAllowed(t *testing.T) {
	t.Run("should return true when GA is in allowed global accounts", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-123", "ga-456", "ga-789"},
		}
		assert.True(t, isGlobalAccountAllowed(config, "ga-123"))
		assert.True(t, isGlobalAccountAllowed(config, "ga-456"))
		assert.True(t, isGlobalAccountAllowed(config, "ga-789"))
	})

	t.Run("should return false when GA is not in allowed global accounts", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-123", "ga-456"},
		}
		assert.False(t, isGlobalAccountAllowed(config, "ga-999"))
		assert.False(t, isGlobalAccountAllowed(config, "ga-000"))
	})

	t.Run("should return true when allowed global accounts contains wildcard", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"*"},
		}
		assert.True(t, isGlobalAccountAllowed(config, "any-ga-id"))
		assert.True(t, isGlobalAccountAllowed(config, "another-ga"))
		assert.True(t, isGlobalAccountAllowed(config, "ga-123"))
	})

	t.Run("should return true when allowed global accounts has both wildcard and specific GAs", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"ga-specific", "*"},
		}
		assert.True(t, isGlobalAccountAllowed(config, "ga-specific"))
		assert.True(t, isGlobalAccountAllowed(config, "any-other-ga"))
	})

	t.Run("should return false when allowed global accounts is empty", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{},
		}
		assert.False(t, isGlobalAccountAllowed(config, "ga-123"))
	})

	t.Run("should return false when config is nil", func(t *testing.T) {
		assert.False(t, isGlobalAccountAllowed(nil, "ga-123"))
	})

	t.Run("should be case-sensitive for GA IDs", func(t *testing.T) {
		config := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{"GA-123"},
		}
		assert.True(t, isGlobalAccountAllowed(config, "GA-123"))
		assert.False(t, isGlobalAccountAllowed(config, "ga-123"))
	})
}

func TestMultiAccountSupport(t *testing.T) {
	rulesService := createRulesService(t)
	stepRetryTuple := internal.RetryTuple{
		Timeout:  2 * time.Second,
		Interval: 1 * time.Second,
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	t.Run("should continue filling most populated binding (fill-most-populated strategy)", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithCredentialsBindings()
		const (
			operationName   = "provisioning-operation-multi-2"
			instanceID      = "instance-multi-2"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// Create 2 existing instances on the same binding (simulating it being the most populated)
		instance1 := fixture.FixInstance("fill-test-1")
		instance1.SubscriptionSecretName = fixture.AWSSecretName
		instance1.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance1))

		instance2 := fixture.FixInstance("fill-test-2")
		instance2.SubscriptionSecretName = fixture.AWSSecretName
		instance2.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance2))

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, fixture.AWSSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should assign new binding when existing one is at full capacity", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithMultipleAWSBindings()
		const (
			operationName   = "provisioning-operation-multi-3"
			instanceID      = "instance-multi-3"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		instance1 := fixture.FixInstance("full-test-1")
		instance1.SubscriptionSecretName = fixture.AWSSecretName
		instance1.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance1))

		instance2 := fixture.FixInstance("full-test-2")
		instance2.SubscriptionSecretName = fixture.AWSSecretName
		instance2.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance2))

		instance3 := fixture.FixInstance("full-test-3")
		instance3.SubscriptionSecretName = fixture.AWSSecretName
		instance3.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance3))

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, fixture.AWSSecretName2, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName2, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should select CB3 when CB1 over limit and CB2 at limit", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithThreeAWSBindings()
		const (
			operationName   = "provisioning-operation-multi-4"
			instanceID      = "instance-multi-4"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 4 instances (over limit of 3)
		for i := 1; i <= 4; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb2-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName2
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB3 has 1 instance (under limit)
		instance1 := fixture.FixInstance("cb3-test-1")
		instance1.SubscriptionSecretName = fixture.AWSSecretName3
		instance1.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance1))

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, fixture.AWSSecretName3, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName3, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should select CB3 when CB1 and CB2 at limit", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithThreeAWSBindings()
		const (
			operationName   = "provisioning-operation-multi-5"
			instanceID      = "instance-multi-5"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-limit-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb2-limit-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName2
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB3 has 0 instances (under limit)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, fixture.AWSSecretName3, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName3, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should select most populated when CB2 and CB3 under limit with equal counts", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithThreeAWSBindings()
		const (
			operationName   = "provisioning-operation-multi-6"
			instanceID      = "instance-multi-6"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-equal-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 1 instance (under limit)
		instance2 := fixture.FixInstance("cb2-equal-test-1")
		instance2.SubscriptionSecretName = fixture.AWSSecretName2
		instance2.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance2))

		// CB3 has 1 instance (under limit, equal to CB2)
		instance3 := fixture.FixInstance("cb3-equal-test-1")
		instance3.SubscriptionSecretName = fixture.AWSSecretName3
		instance3.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance3))

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		// Should select either CB2 or CB3 (both have count=1, so either is valid as "most populated below limit")
		selectedSecret := *operation.ProvisioningParameters.Parameters.TargetSecret
		assert.Contains(t, []string{fixture.AWSSecretName2, fixture.AWSSecretName3}, selectedSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Contains(t, []string{fixture.AWSSecretName2, fixture.AWSSecretName3}, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should select most populated binding below limit (fill-most-populated policy)", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithThreeAWSBindings()
		const (
			operationName   = "provisioning-operation-multi-8"
			instanceID      = "instance-multi-8"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-policy-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 1 instance (under limit, less populated)
		instance2 := fixture.FixInstance("cb2-policy-test-1")
		instance2.SubscriptionSecretName = fixture.AWSSecretName2
		instance2.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance2))

		// CB3 has 2 instances (under limit, more populated than CB2)
		for i := 1; i <= 2; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb3-policy-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName3
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, fixture.AWSSecretName3, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName3, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should claim new binding when all existing bindings are at limit", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithThreeAWSBindingsAndOneUnclaimed()
		const (
			operationName   = "provisioning-operation-multi-7"
			instanceID      = "instance-multi-7"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-claim-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb2-claim-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName2
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB3 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb3-claim-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName3
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, "aws-unclaimed", *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "aws-unclaimed", updatedInstance.SubscriptionSecretName)
	})

	t.Run("should reuse binding after deprovisioning frees space", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithMultipleAWSBindings()
		const (
			operationName   = "provisioning-operation-multi-9"
			instanceID      = "instance-multi-9"
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-deprov-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 3 instances (at limit)
		for i := 1; i <= 3; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb2-deprov-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName2
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// Simulate deprovisioning one instance from CB1
		err := brokerStorage.Instances().Delete("cb1-deprov-test-1")
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(operationName, instanceID, fixture.WithProvider(string(pkg.AWS)))
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation))

		instance := fixture.FixInstance(instanceID)
		instance.SubscriptionSecretName = ""
		instance.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance))

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
		assert.Equal(t, fixture.AWSSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)

		updatedInstance, err := brokerStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName, updatedInstance.SubscriptionSecretName)
	})

	t.Run("should handle multiple sequential provisioning operations until limit reached", func(t *testing.T) {
		// given
		brokerStorage := storage.NewMemoryStorage()
		gardenerClient := fixture.CreateGardenerClientWithThreeAWSBindingsAndOneUnclaimed()
		const (
			platformRegion  = "cf-ap11"
			providerType    = "aws"
			globalAccountID = fixture.AWSTenantName
		)

		// CB1 has 2 instances (one below limit)
		for i := 1; i <= 2; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb1-seq-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB2 has 2 instances (one below limit)
		for i := 1; i <= 2; i++ {
			instance := fixture.FixInstance(fmt.Sprintf("cb2-seq-test-%d", i))
			instance.SubscriptionSecretName = fixture.AWSSecretName2
			instance.GlobalAccountID = globalAccountID
			require.NoError(t, brokerStorage.Instances().Insert(instance))
		}

		// CB3 has 0 instances (empty)

		multiAccountConfig := &multiaccount.MultiAccountConfig{
			AllowedGlobalAccounts: []string{globalAccountID},
			Limits: multiaccount.HyperscalerAccountLimits{
				AWS:     3,
				Default: 100,
			},
		}

		step := NewResolveCredentialsBindingStep(brokerStorage, gardenerClient, rulesService, stepRetryTuple, multiAccountConfig)

		// First provisioning - should select CB1 or CB2 (both have count=2, most populated)
		operation1 := fixture.FixProvisioningOperation("provisioning-operation-seq-1", "instance-seq-1", fixture.WithProvider(string(pkg.AWS)))
		operation1.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation1.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation1.ProvisioningParameters.PlatformRegion = platformRegion
		operation1.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation1))

		instance1 := fixture.FixInstance("instance-seq-1")
		instance1.SubscriptionSecretName = ""
		instance1.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance1))

		operation1, backoff1, err1 := step.Run(operation1, log)
		require.NoError(t, err1)
		assert.Zero(t, backoff1)
		require.NotNil(t, operation1.ProvisioningParameters.Parameters.TargetSecret)
		firstSelected := *operation1.ProvisioningParameters.Parameters.TargetSecret
		assert.Contains(t, []string{fixture.AWSSecretName, fixture.AWSSecretName2}, firstSelected)

		// Update instance with selected binding
		inst1, _ := brokerStorage.Instances().GetByID("instance-seq-1")
		inst1.SubscriptionSecretName = firstSelected
		_, err := brokerStorage.Instances().Update(*inst1)
		require.NoError(t, err)

		// Second provisioning - should select the other one (CB2 or CB1)
		operation2 := fixture.FixProvisioningOperation("provisioning-operation-seq-2", "instance-seq-2", fixture.WithProvider(string(pkg.AWS)))
		operation2.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation2.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation2.ProvisioningParameters.PlatformRegion = platformRegion
		operation2.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation2))

		instance2 := fixture.FixInstance("instance-seq-2")
		instance2.SubscriptionSecretName = ""
		instance2.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance2))

		operation2, backoff2, err2 := step.Run(operation2, log)
		require.NoError(t, err2)
		assert.Zero(t, backoff2)
		require.NotNil(t, operation2.ProvisioningParameters.Parameters.TargetSecret)
		secondSelected := *operation2.ProvisioningParameters.Parameters.TargetSecret
		assert.Contains(t, []string{fixture.AWSSecretName, fixture.AWSSecretName2}, secondSelected)

		// Update instance with selected binding
		inst2, _ := brokerStorage.Instances().GetByID("instance-seq-2")
		inst2.SubscriptionSecretName = secondSelected
		_, err2 = brokerStorage.Instances().Update(*inst2)
		require.NoError(t, err2)

		// Third provisioning - CB1 and CB2 now both at limit (3 each), should select CB3 (has 0)
		operation3 := fixture.FixProvisioningOperation("provisioning-operation-seq-3", "instance-seq-3", fixture.WithProvider(string(pkg.AWS)))
		operation3.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation3.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
		operation3.ProvisioningParameters.PlatformRegion = platformRegion
		operation3.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, brokerStorage.Operations().InsertOperation(operation3))

		instance3 := fixture.FixInstance("instance-seq-3")
		instance3.SubscriptionSecretName = ""
		instance3.GlobalAccountID = globalAccountID
		require.NoError(t, brokerStorage.Instances().Insert(instance3))

		operation3, backoff3, err3 := step.Run(operation3, log)
		require.NoError(t, err3)
		assert.Zero(t, backoff3)
		require.NotNil(t, operation3.ProvisioningParameters.Parameters.TargetSecret)
		thirdSelected := *operation3.ProvisioningParameters.Parameters.TargetSecret
		assert.Equal(t, fixture.AWSSecretName3, thirdSelected)

		updatedInstance3, err := brokerStorage.Instances().GetByID("instance-seq-3")
		require.NoError(t, err)
		assert.Equal(t, fixture.AWSSecretName3, updatedInstance3.SubscriptionSecretName)
	})
}
