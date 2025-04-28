package workers

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process/infrastructure_manager"
	"github.com/kyma-project/kyma-environment-broker/internal/regionssupportingmachine"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestCreateAdditionalWorkers(t *testing.T) {
	t.Run("should create worker with zones from existing worker", func(t *testing.T) {
		// given
		provider := NewProvider(infrastructure_manager.InfrastructureManagerConfig{}, nil, true)
		currentAdditionalWorkers := map[string]gardener.Worker{
			"worker-existing": {
				Name:  "worker-existing",
				Zones: []string{"zone-a", "zone-b", "zone-c"},
			},
		}
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker-existing",
				MachineType: "standard",
				HAZones:     true,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{},
			currentAdditionalWorkers,
			additionalWorkerNodePools,
			[]string{"zone-x", "zone-y", "zone-z"},
			broker.AWSPlanID,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker-existing", workers[0].Name)
		assert.ElementsMatch(t, []string{"zone-a", "zone-b", "zone-c"}, workers[0].Zones)
	})

	t.Run("should create worker with Kyma workload zones", func(t *testing.T) {
		// given
		provider := NewProvider(infrastructure_manager.InfrastructureManagerConfig{}, nil, true)
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker",
				MachineType: "standard",
				HAZones:     true,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.ElementsMatch(t, []string{"zone-a", "zone-b", "zone-c"}, workers[0].Zones)
	})

	t.Run("should create worker with one zone if ha is disabled", func(t *testing.T) {
		// given
		provider := NewProvider(infrastructure_manager.InfrastructureManagerConfig{}, nil, true)
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker",
				MachineType: "standard",
				HAZones:     false,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.Len(t, workers[0].Zones, 1)
		assert.Contains(t, []string{"zone-a", "zone-b", "zone-c"}, workers[0].Zones[0])
	})

	t.Run("should create worker using zones from RegionsSupportingMachine", func(t *testing.T) {
		// given
		regionsSupportingMachine := regionssupportingmachine.RegionsSupportingMachine{
			"standard": {
				"eu-west-1": {"a", "b", "c"},
			},
		}
		provider := NewProvider(infrastructure_manager.InfrastructureManagerConfig{}, regionsSupportingMachine, true)
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker",
				MachineType: "standard",
				HAZones:     true,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{
				Region: "eu-west-1",
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-x", "zone-y", "zone-z"},
			broker.AWSPlanID,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.Len(t, workers[0].Zones, 3)
		assert.ElementsMatch(t, []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"}, workers[0].Zones)
	})

	t.Run("should fail when AvailableZones returns an error", func(t *testing.T) {
		// given
		regionsSupportingMachine := regionssupportingmachine.RegionsSupportingMachine{
			"standard": {
				"eu-west-1": {"a", "b", "c"},
			},
		}
		provider := NewProvider(infrastructure_manager.InfrastructureManagerConfig{}, regionsSupportingMachine, true)
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker",
				MachineType: "standard",
				HAZones:     true,
			},
		}

		// when
		_, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{
				Region: "eu-west-1",
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-x", "zone-y", "zone-z"},
			"not-supported-plan",
		)

		// then
		assert.EqualError(t, err, "while getting available zones from regions supporting machine: plan not-supported-plan not supported")
	})

	t.Run("should skip volume for openstack provider", func(t *testing.T) {
		// given
		provider := NewProvider(infrastructure_manager.InfrastructureManagerConfig{}, nil, true)
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker",
				MachineType: "standard",
				HAZones:     true,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{
				ProviderType: "openstack",
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.SapConvergedCloudPlanID,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.Nil(t, workers[0].Volume)
	})
}
