package provider_test

import (
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	awsProviderName               = "aws"
	gcpProviderName               = "gcp"
	sapConvergedCloudProviderName = "openstack"
	alicloudProviderName          = "alicloud"
	irrelevantMachine             = "irrelevant-machine"
	defaultVolumeSizeGb           = 80
)

type fakePlanConfigProvider struct {
	volumeSizes   map[string]int
	machineTypes  map[string][]string
	hasVolumeSize map[string]bool
}

func newFakeInMemoryPlanConfigProvider() *fakePlanConfigProvider {
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

func (f *fakePlanConfigProvider) withMachineType(planName, machineType string, index ...int) *fakePlanConfigProvider {
	if len(index) != 0 {
		if !f.planMachineTypesListExists(planName) {
			f.createPlanMachineTypesList(planName)
		}
		f.insertMachineTypeAtGivenIndex(planName, machineType, index[0])
	}
	f.machineTypes[planName] = append(f.machineTypes[planName], machineType)
	return f
}

func (f *fakePlanConfigProvider) withVolumeSize(planName string, size int) *fakePlanConfigProvider {
	f.volumeSizes[planName] = size
	f.hasVolumeSize[planName] = true
	return f
}

func (f *fakePlanConfigProvider) planMachineTypesListExists(planName string) bool {
	_, exists := f.machineTypes[planName]
	return exists
}

func (f *fakePlanConfigProvider) createPlanMachineTypesList(planName string) {
	f.machineTypes[planName] = make([]string, 10)
}

func (f *fakePlanConfigProvider) insertMachineTypeAtGivenIndex(planName, machineType string, i int) {
	if len(f.machineTypes[planName]) <= i {
		f.extendMachineTypesListToIndex(planName, i)
	}
	f.machineTypes[planName][i] = machineType
}

func (f *fakePlanConfigProvider) extendMachineTypesListToIndex(planName string, i int) {
	extended := make([]string, i+1)
	copy(extended, f.machineTypes[planName])
	f.machineTypes[planName] = extended
}

func TestPlanSpecificValuesProvider(t *testing.T) {

	t.Run("should return error when plan spec does not contain machine types", func(t *testing.T) {
		// given
		planID := broker.AWSPlanID
		planName := broker.AWSPlanName
		expectedErrMsg := fmt.Sprintf("plan %s (%s) does not contain default machine type", planID, planName)

		params := internal.ProvisioningParameters{
			PlanID: broker.AWSPlanID,
		}

		planConfig := newFakeInMemoryPlanConfigProvider()

		planSpecValProvider := provider.NewPlanSpecificValuesProvider(
			broker.InfrastructureManager{},
			provider.TestTrialPlatformRegionMapping,
			provider.FakeZonesProvider([]string{"a", "b", "c"}),
			planConfig,
		)

		// when
		_, err := planSpecValProvider.ValuesForPlanAndParameters(params)

		// then
		require.Error(t, err)
		assert.Equal(t, expectedErrMsg, err.Error())
	})

	t.Run("AWS provider", func(t *testing.T) {
		changedDefaultMachineType := "m6i.16xlarge"
		changedDefaultVolumeSizeGb := 100

		params := internal.ProvisioningParameters{
			PlanID: broker.AWSPlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
			// given
			planConfig, err := provider.NewFakePlanSpecFromFile()
			require.NoError(t, err)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
				withMachineType(broker.AWSPlanName, changedDefaultMachineType, 0).
				withMachineType(broker.AWSPlanName, provider.DefaultAWSMachineType, 1)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
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

		planConfig := newFakeInMemoryPlanConfigProvider().
			withMachineType(broker.TrialPlanName, irrelevantMachine)

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

		planConfig := newFakeInMemoryPlanConfigProvider().
			withMachineType(broker.FreemiumPlanName, irrelevantMachine)

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
			planConfig, err := provider.NewFakePlanSpecFromFile()
			require.NoError(t, err)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
				withMachineType(broker.AzurePlanName, provider.DefaultAzureMachineType, 1).
				withMachineType(broker.AzurePlanName, changedDefaultMachineType, 0)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
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

		planConfig := newFakeInMemoryPlanConfigProvider().
			withMachineType(broker.TrialPlanName, irrelevantMachine)

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

		planConfig := newFakeInMemoryPlanConfigProvider().
			withMachineType(broker.FreemiumPlanName, irrelevantMachine)

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
			planConfig, err := provider.NewFakePlanSpecFromFile()
			require.NoError(t, err)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
				withMachineType(broker.AzureLitePlanName, changedDefaultMachineType, 0).
				withMachineType(broker.AzureLitePlanName, provider.DefaultOldAzureTrialMachineType, 1)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
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
			planConfig, err := provider.NewFakePlanSpecFromFile()
			require.NoError(t, err)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
				withMachineType(broker.GCPPlanName, changedDefaultMachineType, 0).
				withMachineType(broker.GCPPlanName, provider.DefaultGCPMachineType, 1)

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
			planConfig := newFakeInMemoryPlanConfigProvider().
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

		planConfig := newFakeInMemoryPlanConfigProvider().
			withMachineType(broker.TrialPlanName, irrelevantMachine)

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

	t.Run("SAP Converged Cloud provider", func(t *testing.T) {
		const defaultVolumeSizeGb = 0

		changedDefaultMachineType := "g_c64_m256"

		params := internal.ProvisioningParameters{
			PlanID: broker.SapConvergedCloudPlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
			// given
			planConfig, err := provider.NewFakePlanSpecFromFile()
			require.NoError(t, err)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, sapConvergedCloudProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultSapConvergedCloudMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default machine type", func(t *testing.T) {
			// given
			planConfig := newFakeInMemoryPlanConfigProvider().
				withMachineType(broker.SapConvergedCloudPlanName, changedDefaultMachineType, 0).
				withMachineType(broker.SapConvergedCloudPlanName, provider.DefaultSapConvergedCloudMachineType, 1)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, sapConvergedCloudProviderName, values.ProviderType)
			assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})

	t.Run("Alicloud provider", func(t *testing.T) {
		changedDefaultMachineType := "ecs.g9i.16xlarge"

		params := internal.ProvisioningParameters{
			PlanID: broker.AlicloudPlanID,
		}

		t.Run("should set default values", func(t *testing.T) {
			// given
			planConfig, err := provider.NewFakePlanSpecFromFile()
			require.NoError(t, err)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, alicloudProviderName, values.ProviderType)
			assert.Equal(t, provider.DefaultAlicloudMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})

		t.Run("should change default machine type", func(t *testing.T) {
			// given
			planConfig := newFakeInMemoryPlanConfigProvider().
				withMachineType(broker.AlicloudPlanName, changedDefaultMachineType, 0).
				withMachineType(broker.AlicloudPlanName, provider.DefaultAlicloudMachineType, 1)

			planSpecValProvider := provider.NewPlanSpecificValuesProvider(
				broker.InfrastructureManager{
					MultiZoneCluster: true,
				},
				provider.TestTrialPlatformRegionMapping,
				provider.FakeZonesProvider([]string{"a", "b", "c"}),
				planConfig,
			)

			// when
			values, err := planSpecValProvider.ValuesForPlanAndParameters(params)

			// then
			require.NoError(t, err)
			assert.Equal(t, alicloudProviderName, values.ProviderType)
			assert.Equal(t, changedDefaultMachineType, values.DefaultMachineType)
			assert.Equal(t, defaultVolumeSizeGb, values.VolumeSizeGb)
		})
	})
}
