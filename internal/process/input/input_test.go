package input

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input/automock"
	cloudProvider "github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func fixTrialProviders() []string {
	return []string{"azure", "aws"}
}

func TestInputBuilderFactoryForAzurePlan(t *testing.T) {
	// given
	config := Config{
		URL: "",
	}
	configProvider := mockConfigProvider()

	factory, err := NewInputBuilderFactory(configProvider, config, fixTrialRegionMapping(),
		fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
	assert.NoError(t, err)
	pp := fixProvisioningParameters(broker.AzurePlanID)

	// when
	builder, err := factory.CreateProvisionInput(pp)

	// then
	require.NoError(t, err)

	// when
	shootName := "c-51bcc12"
	builder.
		SetProvisioningParameters(internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Name:         "azure-cluster",
				TargetSecret: ptr.String("azure-secret"),
				Purpose:      ptr.String("development"),
			},
		}).
		SetShootName(shootName).
		SetLabel("label1", "value1").
		SetShootDomain("shoot.domain.sap")
	input, err := builder.CreateProvisionRuntimeInput()
	require.NoError(t, err)
	clusterInput, err := builder.CreateProvisionClusterInput()
	require.NoError(t, err)

	// then
	assert.Equal(t, input.ClusterConfig, clusterInput.ClusterConfig)
	assert.Equal(t, input.RuntimeInput, clusterInput.RuntimeInput)
	assert.Nil(t, clusterInput.KymaConfig)
	assert.Contains(t, input.RuntimeInput.Name, "azure-cluster")
	assert.Equal(t, "azure", input.ClusterConfig.GardenerConfig.Provider)
	assert.Equal(t, "azure-secret", input.ClusterConfig.GardenerConfig.TargetSecret)
	require.NotNil(t, input.ClusterConfig.GardenerConfig.Purpose)
	assert.Equal(t, "development", *input.ClusterConfig.GardenerConfig.Purpose)
	assert.Nil(t, input.ClusterConfig.GardenerConfig.LicenceType)
	assert.Equal(t, shootName, input.ClusterConfig.GardenerConfig.Name)
	assert.NotNil(t, input.ClusterConfig.Administrators)
	assert.Equal(t, gqlschema.Labels{
		"label1": "value1",
	}, input.RuntimeInput.Labels)
}

func TestShouldAdjustRuntimeName(t *testing.T) {
	for name, tc := range map[string]struct {
		runtimeName               string
		expectedNameWithoutSuffix string
	}{
		"regular string": {
			runtimeName:               "test",
			expectedNameWithoutSuffix: "test",
		},
		"too long string": {
			runtimeName:               "this-string-is-too-long-because-it-has-more-than-36-chars",
			expectedNameWithoutSuffix: "this-string-is-too-long-becaus",
		},
		"string with forbidden chars": {
			runtimeName:               "CLUSTER-?name_123@!",
			expectedNameWithoutSuffix: "cluster-name123",
		},
		"too long string with forbidden chars": {
			runtimeName:               "ThisStringIsTooLongBecauseItHasMoreThan36Chars",
			expectedNameWithoutSuffix: "thisstringistoolongbecauseitha",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			configProvider := mockConfigProvider()

			builder, err := NewInputBuilderFactory(configProvider, Config{TrialNodesNumber: 0}, fixTrialRegionMapping(),
				fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
			assert.NoError(t, err)

			pp := fixProvisioningParameters(broker.TrialPlanID)
			pp.Parameters.Name = tc.runtimeName

			creator, err := builder.CreateProvisionInput(pp)
			require.NoError(t, err)
			creator.SetProvisioningParameters(pp)

			// when
			input, err := creator.CreateProvisionRuntimeInput()
			require.NoError(t, err)
			clusterInput, err := creator.CreateProvisionClusterInput()
			require.NoError(t, err)

			// then
			assert.NotEqual(t, pp.Parameters.Name, input.RuntimeInput.Name)
			assert.LessOrEqual(t, len(input.RuntimeInput.Name), 36)
			assert.Equal(t, tc.expectedNameWithoutSuffix, input.RuntimeInput.Name[:len(input.RuntimeInput.Name)-6])
			assert.Equal(t, 1, input.ClusterConfig.GardenerConfig.AutoScalerMin)
			assert.Equal(t, 1, input.ClusterConfig.GardenerConfig.AutoScalerMax)
			assert.Equal(t, tc.expectedNameWithoutSuffix, clusterInput.RuntimeInput.Name[:len(input.RuntimeInput.Name)-6])
			assert.Equal(t, 1, clusterInput.ClusterConfig.GardenerConfig.AutoScalerMin)
			assert.Equal(t, 1, clusterInput.ClusterConfig.GardenerConfig.AutoScalerMax)
		})
	}
}

func TestShouldSetNumberOfNodesForTrialPlan(t *testing.T) {
	// given
	configProvider := mockConfigProvider()

	builder, err := NewInputBuilderFactory(configProvider, Config{TrialNodesNumber: 2},
		fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
	assert.NoError(t, err)

	pp := fixProvisioningParameters(broker.TrialPlanID)

	creator, err := builder.CreateProvisionInput(pp)
	require.NoError(t, err)
	creator.SetProvisioningParameters(pp)

	// when
	input, err := creator.CreateProvisionRuntimeInput()
	require.NoError(t, err)
	clusterInput, err := creator.CreateProvisionClusterInput()
	require.NoError(t, err)

	// then
	assert.Equal(t, 2, input.ClusterConfig.GardenerConfig.AutoScalerMin)
	assert.Equal(t, 2, clusterInput.ClusterConfig.GardenerConfig.AutoScalerMax)
}

func TestShouldSetGlobalConfiguration(t *testing.T) {
	t.Run("When creating ProvisionRuntimeInput", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		builder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.TrialPlanID)

		creator, err := builder.CreateProvisionInput(pp)
		require.NoError(t, err)
		creator.SetProvisioningParameters(pp)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		expectedStrategy := gqlschema.ConflictStrategyReplace
		assert.Equal(t, &expectedStrategy, input.KymaConfig.ConflictStrategy)
	})

	t.Run("When creating UpgradeRuntimeInput", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		builder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.TrialPlanID)

		creator, err := builder.CreateUpgradeInput(pp)
		require.NoError(t, err)
		creator.SetProvisioningParameters(pp)

		// when
		input, err := creator.CreateUpgradeRuntimeInput()
		require.NoError(t, err)

		// then
		expectedStrategy := gqlschema.ConflictStrategyReplace
		assert.Equal(t, &expectedStrategy, input.KymaConfig.ConflictStrategy)
	})
}

func TestCreateProvisionRuntimeInput_ConfigureDNS(t *testing.T) {

	t.Run("should apply provided DNS Providers values", func(t *testing.T) {
		// given
		expectedDnsValues := &gqlschema.DNSConfigInput{
			Domain: "shoot-name.domain.sap",
			Providers: []*gqlschema.DNSProviderInput{
				{
					DomainsInclude: []string{"devtest.kyma.ondemand.com"},
					Primary:        true,
					SecretName:     "aws_dns_domain_secrets_test_incustom",
					Type:           "route53_type_test",
				},
			},
		}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(),
			fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedDnsValues, input.ClusterConfig.GardenerConfig.DNSConfig)
		assert.Equal(t, expectedDnsValues, clusterInput.ClusterConfig.GardenerConfig.DNSConfig)
	})

	t.Run("should apply the DNS Providers values while DNS providers is empty", func(t *testing.T) {
		// given
		expectedDnsValues := &gqlschema.DNSConfigInput{
			Domain: "shoot-name.domain.sap",
		}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)
		setRuntimeProperties(creator)
		creator.SetShootDNSProviders(gardener.DNSProvidersData{})

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedDnsValues, input.ClusterConfig.GardenerConfig.DNSConfig)
		assert.Equal(t, expectedDnsValues, clusterInput.ClusterConfig.GardenerConfig.DNSConfig)
	})

}

func TestCreateProvisionRuntimeInput_ConfigureOIDC(t *testing.T) {

	t.Run("should apply default OIDC values when OIDC is nil", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
			GroupsClaim:    "groups",
			IssuerURL:      "https://kymatest.accounts400.ondemand.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "sub",
			UsernamePrefix: "-",
		}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})

	t.Run("should apply default OIDC values when all OIDC fields are empty", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
			GroupsClaim:    "groups",
			IssuerURL:      "https://kymatest.accounts400.ondemand.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "sub",
			UsernamePrefix: "-",
		}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.OIDC = &pkg.OIDCConfigDTO{}

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})

	t.Run("should apply provided OIDC values", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "provided-id",
			GroupsClaim:    "fake-groups-claim",
			IssuerURL:      "https://test.domain.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "usernameClaim",
			UsernamePrefix: "<<",
		}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.OIDC = &pkg.OIDCConfigDTO{
			ClientID:       "provided-id",
			GroupsClaim:    "fake-groups-claim",
			IssuerURL:      "https://test.domain.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "usernameClaim",
			UsernamePrefix: "<<",
		}

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})
}

func TestCreateProvisionRuntimeInput_ConfigureAdmins(t *testing.T) {
	t.Run("should apply default admin from user_id field", func(t *testing.T) {
		// given
		expectedAdmins := []string{"fake-user-id"}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.ErsContext.UserID = expectedAdmins[0]

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.ClusterConfig.Administrators)
	})

	t.Run("should apply new admin list", func(t *testing.T) {
		// given
		expectedAdmins := []string{"newAdmin1@test.com", "newAdmin2@test.com"}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.RuntimeAdministrators = expectedAdmins

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams)
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.ClusterConfig.Administrators)
	})
}

func setRuntimeProperties(creator internal.ProvisionerInputCreator) {
	creator.SetKubeconfig("example kubeconfig payload")
	creator.SetRuntimeID("runtimeID")
	creator.SetInstanceID("instanceID")
	creator.SetShootName("shoot-name")
	creator.SetShootDomain("shoot-name.domain.sap")
	creator.SetShootDNSProviders(fixture.FixDNSProvidersConfig())
}

func TestCreateUpgradeRuntimeInput_ConfigureAdmins(t *testing.T) {
	t.Run("should not overwrite default admin (from user_id)", func(t *testing.T) {
		// given
		expectedAdmins := []string{"fake-user-id"}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.ErsContext.UserID = expectedAdmins[0]

		creator, err := inputBuilder.CreateUpgradeShootInput(provisioningParams)
		require.NoError(t, err)

		// when
		creator.SetProvisioningParameters(provisioningParams)
		input, err := creator.CreateUpgradeShootInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.Administrators)
	})

	t.Run("should overwrite default admin with new admins list", func(t *testing.T) {
		// given
		userId := "fake-user-id"
		expectedAdmins := []string{"newAdmin1@test.com", "newAdmin2@test.com"}

		id := uuid.New().String()

		configProvider := mockConfigProvider()

		inputBuilder, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.ErsContext.UserID = userId
		provisioningParams.Parameters.RuntimeAdministrators = expectedAdmins

		creator, err := inputBuilder.CreateUpgradeShootInput(provisioningParams)
		require.NoError(t, err)

		// when
		creator.SetProvisioningParameters(provisioningParams)
		input, err := creator.CreateUpgradeShootInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.Administrators)
	})
}

func TestCreateUpgradeShootInput_ConfigureAutoscalerParams(t *testing.T) {
	t.Run("should not apply CreateUpgradeShootInput with provisioning autoscaler parameters", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		//ar provider HyperscalerInputProvider

		pp := fixProvisioningParameters(broker.GCPPlanID)
		//provider = &cloudProvider.GcpInput{} // for broker.GCPPlanID

		rtinput, err := ibf.CreateUpgradeShootInput(pp)

		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, rtinput)

		rtinput = rtinput.SetProvisioningParameters(pp)
		input, err := rtinput.CreateUpgradeShootInput()
		assert.NoError(t, err)

		expectMaxSurge := *pp.Parameters.MaxSurge
		expectMaxUnavailable := *pp.Parameters.MaxUnavailable
		t.Logf("%v, %v", expectMaxSurge, expectMaxUnavailable)

		assert.Nil(t, input.GardenerConfig.AutoScalerMin)
		assert.Nil(t, input.GardenerConfig.AutoScalerMax)
		assert.Equal(t, expectMaxSurge, *input.GardenerConfig.MaxSurge)
		assert.Equal(t, expectMaxUnavailable, *input.GardenerConfig.MaxUnavailable)
	})

	t.Run("should not apply CreateUpgradeShootInput with provider autoscaler parameters", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.GCPPlanID)
		pp.Parameters.AutoScalerMin = nil
		pp.Parameters.AutoScalerMax = nil
		pp.Parameters.MaxSurge = nil
		pp.Parameters.MaxUnavailable = nil

		provider := &cloudProvider.GcpInput{} // for broker.GCPPlanID

		rtinput, err := ibf.CreateUpgradeShootInput(pp)

		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, rtinput)

		rtinput = rtinput.SetProvisioningParameters(pp)
		input, err := rtinput.CreateUpgradeShootInput()
		assert.NoError(t, err)

		expectMaxSurge := provider.Defaults().GardenerConfig.MaxSurge
		expectMaxUnavailable := provider.Defaults().GardenerConfig.MaxUnavailable

		assert.Nil(t, input.GardenerConfig.AutoScalerMin)
		assert.Nil(t, input.GardenerConfig.AutoScalerMax)
		assert.Equal(t, expectMaxSurge, *input.GardenerConfig.MaxSurge)
		assert.Equal(t, expectMaxUnavailable, *input.GardenerConfig.MaxUnavailable)
	})
}

func TestShootAndSeedSameRegion(t *testing.T) {

	t.Run("should set shootAndSeedSameRegion field on provisioner input if feature flag is enabled", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		builder, err := NewInputBuilderFactory(configProvider, Config{EnableShootAndSeedSameRegion: true},
			fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)

		pp := fixture.FixProvisioningParameters("")
		pp.Parameters.ShootAndSeedSameRegion = ptr.Bool(true)

		// when
		creator, err := builder.CreateProvisionInput(pp)
		require.NoError(t, err)
		input, err := creator.CreateProvisionRuntimeInput()

		// then
		require.NoError(t, err)
		assert.NotNil(t, input.ClusterConfig.GardenerConfig.ShootAndSeedSameRegion)
		assert.True(t, *input.ClusterConfig.GardenerConfig.ShootAndSeedSameRegion)
	})
}

func mockConfigProvider() ConfigurationProvider {
	configProvider := &automock.ConfigurationProvider{}
	configProvider.On("ProvideForGivenPlan",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).
		Return(&internal.ConfigForPlan{}, nil)
	return configProvider
}
