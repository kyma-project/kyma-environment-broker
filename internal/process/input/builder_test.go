package input

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	cloudProvider "github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputBuilderFactory_IsPlanSupport(t *testing.T) {
	// given
	configProvider := mockConfigProvider()

	ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
	assert.NoError(t, err)

	// when/then
	assert.True(t, ibf.IsPlanSupport(broker.GCPPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzurePlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AWSPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.TrialPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.FreemiumPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.BuildRuntimeAWSPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.BuildRuntimeAzurePlanID))
	assert.True(t, ibf.IsPlanSupport(broker.BuildRuntimeGCPPlanID))
}

func TestInputBuilderFactory_ForPlan(t *testing.T) {
	t.Run("should build RuntimeInput with default version Kyma components and ProvisionRuntimeInput", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID)

		// when
		input, err := ibf.CreateProvisionInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.provisionRuntimeInput)
		assert.Nil(t, result.upgradeRuntimeInput.KymaConfig)

	})

	t.Run("should build RuntimeInput with default version Kyma components and UpgradeRuntimeInput", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID)

		// when
		input, err := ibf.CreateUpgradeInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
	})

	t.Run("should build RuntimeInput with set version Kyma components", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID)

		// when
		input, err := ibf.CreateProvisionInput(pp)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, &RuntimeInput{}, input)
	})

	t.Run("should build RuntimeInput with proper plan", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID)

		// when
		input, err := ibf.CreateProvisionInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.Equal(t, gqlschema.KymaProfileProduction, *result.provisionRuntimeInput.KymaConfig.Profile)

		// given
		pp = fixProvisioningParameters(broker.TrialPlanID)

		// when
		input, err = ibf.CreateProvisionInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result = input.(*RuntimeInput)
		assert.Equal(t, gqlschema.KymaProfileEvaluation, *result.provisionRuntimeInput.KymaConfig.Profile)

	})

	t.Run("should build UpgradeRuntimeInput with proper profile", func(t *testing.T) {
		// given
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID)

		// when
		input, err := ibf.CreateUpgradeInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput.KymaConfig.Profile)
		assert.Equal(t, gqlschema.KymaProfileProduction, *result.upgradeRuntimeInput.KymaConfig.Profile)

		// given
		pp = fixProvisioningParameters(broker.TrialPlanID)
		provider := pkg.GCP
		pp.Parameters.Provider = &provider
		// when
		input, err = ibf.CreateUpgradeInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result = input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput.KymaConfig.Profile)
		assert.Equal(t, gqlschema.KymaProfileEvaluation, *result.upgradeRuntimeInput.KymaConfig.Profile)
	})

	t.Run("should build CreateUpgradeShootInput with proper autoscaler parameters", func(t *testing.T) {
		// given
		var provider HyperscalerInputProvider
		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(configProvider, Config{}, fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO(), false)
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID)
		provider = &cloudProvider.GcpInput{} // for broker.GCPPlanID

		// when
		input, err := ibf.CreateUpgradeShootInput(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		maxSurge := *result.upgradeShootInput.GardenerConfig.MaxSurge
		maxUnavailable := *result.upgradeShootInput.GardenerConfig.MaxUnavailable

		assert.Nil(t, result.upgradeShootInput.GardenerConfig.AutoScalerMax)
		assert.Nil(t, result.upgradeShootInput.GardenerConfig.AutoScalerMin)
		assert.Equal(t, maxSurge, provider.Defaults().GardenerConfig.MaxSurge)
		assert.Equal(t, maxUnavailable, provider.Defaults().GardenerConfig.MaxUnavailable)
		t.Logf("%v, %v", maxSurge, maxUnavailable)
	})

}

func fixProvisioningParameters(planID string) internal.ProvisioningParameters {
	pp := fixture.FixProvisioningParameters("")
	pp.PlanID = planID
	pp.Parameters.AutoScalerMin = ptr.Integer(1)
	pp.Parameters.AutoScalerMax = ptr.Integer(1)

	return pp
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}
