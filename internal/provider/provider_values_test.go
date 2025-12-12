package provider_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	awsProviderName     = "aws"
	gcpProviderName     = "gcp"
	unrelevantMachine   = "unrelevant-machine"
	defaultVolumeSizeGb = 80
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

func TestPlanSpecificValuesProvider(t *testing.T) {

	t.Run("AWS provider", func(t *testing.T) {
		changedDefaultMachineType := "m6i.16xlarge"
		changedDefaultVolumeSizeGb := 100

		params := internal.ProvisioningParameters{
			PlanID: broker.AWSPlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
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

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultAWSMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default machine type", func(t *testing.T) {
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

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default volume size", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AWSPlanName, provider.DefaultAWSMachineType).
				withVolumeSize(broker.AWSPlanName, changedDefaultVolumeSizeGb)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultAWSMachineType, values.DefaultMachineType)
			assert.Equal(t, changedDefaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("AWS trial provider", func(t *testing.T) {
		const defaultVolumeSizeGb = 50

		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.TrialPlanName, unrelevantMachine)

		params := internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
		}

		t.Run("should set default values with bigger machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: false,
					DefaultTrialProvider:   runtime.AWS,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultOldAWSTrialMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should set default values with smaller machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: true,
					DefaultTrialProvider:   runtime.AWS,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultAWSMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("AWS free provider", func(t *testing.T) {
		const defaultVolumeSizeGb = 50

		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.FreemiumPlanName, unrelevantMachine)

		params := internal.ProvisioningParameters{
			PlanID:           broker.FreemiumPlanID,
			PlatformProvider: runtime.AWS,
		}

		t.Run("should set default values with bigger machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: false,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultOldAWSTrialMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should set default values with smaller machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, awsProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultAWSMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("Azure provider", func(t *testing.T) {
		changedDefaultMachineType := "Standard_D64s_v5"
		changedDefaultVolumeSizeGb := 100

		params := internal.ProvisioningParameters{
			PlanID: broker.AzurePlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AzurePlanName, provider.DefaultAzureMachineType).
				withMachineType(broker.AzurePlanName, changedDefaultMachineType)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultAzureMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default machine type", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AzurePlanName, changedDefaultMachineType).
				withMachineType(broker.AzurePlanName, provider.DefaultAzureMachineType)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default volume size", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AzurePlanName, provider.DefaultAzureMachineType).
				withVolumeSize(broker.AzurePlanName, changedDefaultVolumeSizeGb)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultAzureMachineType, values.DefaultMachineType)
			assert.Equal(t, changedDefaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("Azure trial provider", func(t *testing.T) {
		const defaultVolumeSizeGb = 50

		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.TrialPlanName, unrelevantMachine)

		params := internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
		}

		t.Run("should set default values with bigger machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: false,
					DefaultTrialProvider:   runtime.Azure,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultOldAzureTrialMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should set default values with smaller machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: true,
					DefaultTrialProvider:   runtime.Azure,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultAzureMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("Azure free provider", func(t *testing.T) {
		const defaultVolumeSizeGb = 50

		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.FreemiumPlanName, unrelevantMachine)

		params := internal.ProvisioningParameters{
			PlanID:           broker.FreemiumPlanID,
			PlatformProvider: runtime.Azure,
		}

		t.Run("should set default values with bigger machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: false,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultOldAzureTrialMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should set default values with smaller machine type", func(t *testing.T) {
			// given
			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					UseSmallerMachineTypes: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultAzureMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("Azure Lite provider", func(t *testing.T) {
		changedDefaultMachineType := "Standard_D64s_v5"
		changedDefaultVolumeSizeGb := 100

		params := internal.ProvisioningParameters{
			PlanID: broker.AzureLitePlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AzureLitePlanName, provider.DefaultOldAzureTrialMachineType).
				withMachineType(broker.AzureLitePlanName, changedDefaultMachineType)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultOldAzureTrialMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default machine type", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AzureLitePlanName, changedDefaultMachineType).
				withMachineType(broker.AzureLitePlanName, provider.DefaultOldAzureTrialMachineType)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default volume size", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.AzureLitePlanName, provider.DefaultOldAzureTrialMachineType).
				withVolumeSize(broker.AzureLitePlanName, changedDefaultVolumeSizeGb)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"1", "2", "3"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, "azure", values.ProviderType)
			assert.Equal(t, provider.DefaultOldAzureTrialMachineType, values.DefaultMachineType)
			assert.Equal(t, changedDefaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("GCP provider", func(t *testing.T) {
		changedDefaultMachineType := "n2-standard-64"
		changedDefaultVolumeSizeGb := 100

		params := internal.ProvisioningParameters{
			PlanID: broker.GCPPlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.GCPPlanName, provider.DefaultGCPMachineType).
				withMachineType(broker.GCPPlanName, changedDefaultMachineType)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, gcpProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultGCPMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default machine type", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.GCPPlanName, changedDefaultMachineType).
				withMachineType(broker.GCPPlanName, provider.DefaultGCPMachineType)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, gcpProviderName, values.ProviderType)
			assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default volume size", func(t *testing.T) {
			// given
			planConfig := newFakePlanConfigProvider().
				withMachineType(broker.GCPPlanName, provider.DefaultGCPMachineType).
				withVolumeSize(broker.GCPPlanName, changedDefaultVolumeSizeGb)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, gcpProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultGCPMachineType, values.DefaultMachineType)
			assert.Equal(t, changedDefaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("GCP trial provider", func(t *testing.T) {
		// given
		const defaultVolumeSizeGb = 30

		planConfig := newFakePlanConfigProvider().
			withMachineType(broker.TrialPlanName, unrelevantMachine)

		planSpecValProvider := provider.NewPlanSpecificValuesProvider(
			broker.InfrastructureManager{
				DefaultTrialProvider: runtime.GCP,
			},
			provider.TestTrialPlatformRegionMapping,
			provider.FakeZonesProvider([]string{"a", "b", "c"}),
			planConfig,
		)

		params := internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
		}

		// when
		values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

		// then
		require.NoError(t, err)
		assert.Equal(t, gcpProviderName, values.ProviderType)
		assert.Equal(t, provider.DefaultGCPTrialMachineType, values.DefaultMachineType)
		assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
	})
}
