package provider_test

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePlanConfigProvider struct {
	volumeSizes   map[string]int
	machineTypes  map[string]string
	hasVolumeSize map[string]bool
}

func newFakePlanConfigProvider() *fakePlanConfigProvider {
	return &fakePlanConfigProvider{
		volumeSizes:   make(map[string]int),
		machineTypes:  make(map[string]string),
		hasVolumeSize: make(map[string]bool),
	}
}

func (f *fakePlanConfigProvider) DefaultVolumeSizeGb(planName string) (int, bool) {
	size, ok := f.volumeSizes[planName]
	if !ok {
		return 0, false
	}
	return size, f.hasVolumeSize[planName]
}

func (f *fakePlanConfigProvider) DefaultMachineType(planName string) string {
	return f.machineTypes[planName]
}

func (f *fakePlanConfigProvider) withMachineType(planName, machineType string) *fakePlanConfigProvider {
	f.machineTypes[planName] = machineType
	return f
}

func (f *fakePlanConfigProvider) withVolumeSize(planName string, size int) *fakePlanConfigProvider {
	f.volumeSizes[planName] = size
	f.hasVolumeSize[planName] = true
	return f
}

func TestPlanSpecificValuesProvider_AWSPlan(t *testing.T) {

	t.Run("default values", func(t *testing.T) {
		// given
		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.AWSPlanName, provider.DefaultAWSMachineType)

		planSpecValProvider := provider.NewPlanSpecificValuesProvider(
			broker.InfrastructureManager{},
			provider.TestTrialPlatformRegionMapping,
			provider.FakeZonesProvider([]string{"a", "b", "c"}),
			planConfig,
		)

		params := internal.ProvisioningParameters{
			PlanID:         broker.AWSPlanID,
			Parameters:     pkg.ProvisioningParametersDTO{},
			PlatformRegion: "cf-eu10",
		}

		// when
		values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

		// then
		require.NoError(t, err)
		assert.Equal(t, "aws", values.ProviderType)
		assert.Equal(t, provider.DefaultAWSMachineType, values.DefaultMachineType)
		assert.Equal(t, 80, values.VolumeSizeGb)
	})
}
