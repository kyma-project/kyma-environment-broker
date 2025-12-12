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
	machineTypes  map[string][]string
	hasVolumeSize map[string]bool
}

func newFakePlanConfigProvider() *fakePlanConfigProvider {
	return &fakePlanConfigProvider{
		volumeSizes:   make(map[string]int),
		machineTypes:  make(map[string][]string),
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
	machineTypes, ok := f.machineTypes[planName]
	if !ok {
		return ""
	}
	return machineTypes[0]
}

func (f *fakePlanConfigProvider) withMachineType(planName, machineType string) *fakePlanConfigProvider {
	f.machineTypes[planName] = append(f.machineTypes[planName], machineType)
	return f
}

func (f *fakePlanConfigProvider) withVolumeSize(planName string, size int) *fakePlanConfigProvider {
	f.volumeSizes[planName] = size
	f.hasVolumeSize[planName] = true
	return f
}

func TestPlanSpecificValuesProvider_AWSPlan(t *testing.T) {
	var changedDefaultMachineType = "m6i.16xlarge"

	t.Run("default values", func(t *testing.T) {
		// given
		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.AWSPlanName, provider.DefaultAWSMachineType).
			withMachineType(broker.AWSPlanName, changedDefaultMachineType)

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

	t.Run("changed default machine type", func(t *testing.T) {
		// given
		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.AWSPlanName, changedDefaultMachineType).
			withMachineType(broker.AWSPlanName, provider.DefaultAWSMachineType)

		planSpecValProvider := provider.NewPlanSpecificValuesProvider(
			broker.InfrastructureManager{},
			provider.TestTrialPlatformRegionMapping,
			provider.FakeZonesProvider([]string{"a", "b", "c"}),
			planConfig,
		)

		params := internal.ProvisioningParameters{
			PlanID: broker.AWSPlanID,
			Parameters: pkg.ProvisioningParametersDTO{
				MachineType: &changedDefaultMachineType,
			},
			PlatformRegion: "cf-eu10",
		}

		// when
		values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

		// then
		require.NoError(t, err)
		assert.Equal(t, "aws", values.ProviderType)
		assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
		assert.Equal(t, 80, values.VolumeSizeGb)
	})
}
