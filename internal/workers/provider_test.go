package workers

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	provider2 "github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateAdditionalWorkers(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("should create worker with zones from existing worker", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
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
			internal.ProviderValues{ProviderType: provider2.AWSProviderType},
			currentAdditionalWorkers,
			additionalWorkerNodePools,
			[]string{"zone-x", "zone-y", "zone-z"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker-existing", workers[0].Name)
		assert.ElementsMatch(t, []string{"zone-a", "zone-b", "zone-c"}, workers[0].Zones)
	})

	t.Run("should create worker with Kyma workload zones", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
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
				ProviderType: provider2.AWSProviderType,
				VolumeSizeGb: 115,
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.ElementsMatch(t, []string{"zone-a", "zone-b", "zone-c"}, workers[0].Zones)
		assert.Equal(t, "115Gi", workers[0].Volume.VolumeSize)
	})

	t.Run("should create worker with one zone if ha is disabled", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker",
				MachineType: "standard",
				HAZones:     false,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{ProviderType: provider2.AWSProviderType},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.Len(t, workers[0].Zones, 1)
		assert.Contains(t, []string{"zone-a", "zone-b", "zone-c"}, workers[0].Zones[0])
	})

	t.Run("should create worker using zones from RegionsSupportingMachine", func(t *testing.T) {
		providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  regionsSupportingMachine:
    standard:
      eu-west-1: [a, b, c]
`))
		assert.NoError(t, err)
		provider := NewProvider(broker.InfrastructureManager{}, providerSpec)
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
				Region:       "eu-west-1",
				ProviderType: provider2.AWSProviderType,
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-x", "zone-y", "zone-z"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.Len(t, workers[0].Zones, 3)
		assert.ElementsMatch(t, []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"}, workers[0].Zones)
	})

	t.Run("should skip volume for openstack provider", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
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
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker", workers[0].Name)
		assert.Nil(t, workers[0].Volume)
	})

	t.Run("should use discovered zones", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, fixture.NewProviderSpecWithZonesDiscovery(t, true))
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "worker-1",
				MachineType: "m6i.large",
				HAZones:     true,
			},
			{
				Name:        "worker-2",
				MachineType: "m6i.large",
				HAZones:     false,
			},
			{
				Name:        "worker-3",
				MachineType: "m5.large",
				HAZones:     false,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{
				ProviderType: "aws",
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
			map[string][]string{
				"m6i.large": {"zone-d", "zone-e", "zone-f", "zone-h"},
				"m5.large":  {"zone-i", "zone-j"},
			},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 3)
		assertWorker(t, workers, "worker-1", 3, "zone-d", "zone-e", "zone-f", "zone-h")
		assertWorker(t, workers, "worker-2", 1, "zone-d", "zone-e", "zone-f", "zone-h")
		assertWorker(t, workers, "worker-3", 1, "zone-i", "zone-j")
	})

	t.Run("should map taints to gardener worker", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:          "worker-tainted",
				MachineType:   "standard",
				HAZones:       true,
				AutoScalerMin: 3,
				AutoScalerMax: 10,
				Taints: []runtime.TaintDTO{
					{Key: "gpu", Value: "true", Effect: runtime.TaintEffectNoSchedule},
					{Key: "dedicated", Value: "ml", Effect: runtime.TaintEffectPreferNoSchedule},
				},
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{
				ProviderType: provider2.AWSProviderType,
				VolumeSizeGb: 115,
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "worker-tainted", workers[0].Name)
		assert.Len(t, workers[0].Taints, 2)
		assert.Equal(t, corev1.Taint{Key: "gpu", Value: "true", Effect: corev1.TaintEffectNoSchedule}, workers[0].Taints[0])
		assert.Equal(t, corev1.Taint{Key: "dedicated", Value: "ml", Effect: corev1.TaintEffectPreferNoSchedule}, workers[0].Taints[1])
	})

	t.Run("should set CRI with gvisor when gvisor is enabled", func(t *testing.T) {
		// given
		p := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "gvisor-pool",
				MachineType: "standard",
				HAZones:     true,
				Gvisor:      &runtime.GvisorDTO{Enabled: true},
			},
		}

		// when
		workers, err := p.CreateAdditionalWorkers(
			internal.ProviderValues{ProviderType: provider2.AWSProviderType},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		require.Len(t, workers, 1)
		require.NotNil(t, workers[0].CRI)
		assert.Equal(t, gardener.CRINameContainerD, workers[0].CRI.Name)
		assert.Equal(t, []gardener.ContainerRuntime{{Type: "gvisor"}}, workers[0].CRI.ContainerRuntimes)
	})

	t.Run("should not set CRI when gvisor is nil", func(t *testing.T) {
		// given
		p := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "no-gvisor-pool",
				MachineType: "standard",
				HAZones:     true,
				Gvisor:      nil,
			},
		}

		// when
		workers, err := p.CreateAdditionalWorkers(
			internal.ProviderValues{ProviderType: provider2.AWSProviderType},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		require.Len(t, workers, 1)
		assert.Nil(t, workers[0].CRI)
	})

	t.Run("should not set CRI when gvisor is disabled", func(t *testing.T) {
		// given
		p := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:        "gvisor-disabled-pool",
				MachineType: "standard",
				HAZones:     true,
				Gvisor:      &runtime.GvisorDTO{Enabled: false},
			},
		}

		// when
		workers, err := p.CreateAdditionalWorkers(
			internal.ProviderValues{ProviderType: provider2.AWSProviderType},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		require.Len(t, workers, 1)
		assert.Nil(t, workers[0].CRI)
	})

	t.Run("should not set taints when none provided", func(t *testing.T) {
		// given
		provider := NewProvider(broker.InfrastructureManager{}, newEmptyProviderSpec())
		additionalWorkerNodePools := []runtime.AdditionalWorkerNodePool{
			{
				Name:          "worker-no-taints",
				MachineType:   "standard",
				HAZones:       true,
				AutoScalerMin: 3,
				AutoScalerMax: 10,
			},
		}

		// when
		workers, err := provider.CreateAdditionalWorkers(
			internal.ProviderValues{
				ProviderType: provider2.AWSProviderType,
				VolumeSizeGb: 115,
			},
			nil,
			additionalWorkerNodePools,
			[]string{"zone-a", "zone-b", "zone-c"},
			broker.AWSPlanID,
			map[string][]string{},
			&internal.Operation{
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{},
				},
			},
			log,
		)

		// then
		assert.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Nil(t, workers[0].Taints)
	})
}

func TestToGardenerTaints(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := toGardenerTaints(nil)
		assert.Nil(t, result)
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		result := toGardenerTaints([]runtime.TaintDTO{})
		assert.Nil(t, result)
	})

	t.Run("single taint is mapped correctly", func(t *testing.T) {
		taints := []runtime.TaintDTO{
			{Key: "dedicated", Value: "gpu", Effect: runtime.TaintEffectNoSchedule},
		}
		result := toGardenerTaints(taints)
		assert.Equal(t, []corev1.Taint{
			{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
		}, result)
	})

	t.Run("same key with different effects are mapped", func(t *testing.T) {
		taints := []runtime.TaintDTO{
			{Key: "dedicated", Value: "gpu", Effect: runtime.TaintEffectNoSchedule},
			{Key: "dedicated", Value: "gpu", Effect: runtime.TaintEffectNoExecute},
		}
		result := toGardenerTaints(taints)
		assert.Equal(t, []corev1.Taint{
			{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
			{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoExecute},
		}, result)
	})

	t.Run("multiple taints are all mapped", func(t *testing.T) {
		taints := []runtime.TaintDTO{
			{Key: "k1", Value: "v1", Effect: runtime.TaintEffectNoSchedule},
			{Key: "k2", Value: "v2", Effect: runtime.TaintEffectPreferNoSchedule},
			{Key: "k3", Value: "", Effect: runtime.TaintEffectNoExecute},
		}
		result := toGardenerTaints(taints)
		assert.Equal(t, []corev1.Taint{
			{Key: "k1", Value: "v1", Effect: corev1.TaintEffectNoSchedule},
			{Key: "k2", Value: "v2", Effect: corev1.TaintEffectPreferNoSchedule},
			{Key: "k3", Value: "", Effect: corev1.TaintEffectNoExecute},
		}, result)
	})

	t.Run("taint without value is mapped with empty value", func(t *testing.T) {
		taints := []runtime.TaintDTO{
			{Key: "special", Effect: runtime.TaintEffectNoExecute},
		}
		result := toGardenerTaints(taints)
		assert.Equal(t, []corev1.Taint{
			{Key: "special", Value: "", Effect: corev1.TaintEffectNoExecute},
		}, result)
	})
}

func TestResolveMachineType(t *testing.T) {
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
  machinesVersions:
    "mi.{size}": "m7i.{size}"
`))
	require.NoError(t, err)

	provider := NewProvider(broker.InfrastructureManager{}, providerSpec)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tests := []struct {
		name                     string
		operation                *internal.Operation
		additionalWorkerNodePool runtime.AdditionalWorkerNodePool
		workerExists             bool
		currentAdditionalWorker  gardener.Worker
		want                     string
	}{
		{
			name: "provision resolves machine type from provider spec",
			operation: &internal.Operation{
				Type: internal.OperationTypeProvision,
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{
						ProviderType: "aws",
					},
				},
			},
			additionalWorkerNodePool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "mi.large",
			},
			workerExists: false,
			want:         "m7i.large",
		},
		{
			name: "update reuses existing machine type when worker exists and pool is unchanged",
			operation: &internal.Operation{
				Type: internal.OperationTypeUpdate,
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{
						ProviderType: "aws",
					},
				},
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "mi.large",
							},
						},
					},
				},
			},
			additionalWorkerNodePool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "mi.large",
			},
			workerExists: true,
			currentAdditionalWorker: gardener.Worker{
				Machine: gardener.Machine{
					Type: "m6i.large",
				},
			},
			want: "m6i.large",
		},
		{
			name: "update resolves machine type when worker exists but pool changed",
			operation: &internal.Operation{
				Type: internal.OperationTypeUpdate,
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{
						ProviderType: "aws",
					},
				},
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "mi.large",
							},
						},
					},
				},
			},
			additionalWorkerNodePool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "mi.xlarge",
			},
			workerExists: true,
			currentAdditionalWorker: gardener.Worker{
				Machine: gardener.Machine{
					Type: "m6i.large",
				},
			},
			want: "m7i.xlarge",
		},
		{
			name: "update resolves machine type when worker does not exist",
			operation: &internal.Operation{
				Type: internal.OperationTypeUpdate,
				InstanceDetails: internal.InstanceDetails{
					ProviderValues: &internal.ProviderValues{
						ProviderType: "aws",
					},
				},
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "mi.large",
							},
						},
					},
				},
			},
			additionalWorkerNodePool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "mi.large",
			},
			workerExists: false,
			want:         "m7i.large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.ResolveMachineType(
				tt.operation,
				tt.additionalWorkerNodePool,
				tt.workerExists,
				tt.currentAdditionalWorker,
				log,
			)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsAdditionalWorkerPoolUnchanged(t *testing.T) {
	tests := []struct {
		name                 string
		operation            *internal.Operation
		additionalWorkerPool runtime.AdditionalWorkerNodePool
		want                 bool
	}{
		{
			name: "returns true when matching name and machine type exists",
			operation: &internal.Operation{
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "m5.large",
							},
						},
					},
				},
			},
			additionalWorkerPool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "m5.large",
			},
			want: true,
		},
		{
			name: "returns false when name matches but machine type differs",
			operation: &internal.Operation{
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "m5.large",
							},
						},
					},
				},
			},
			additionalWorkerPool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "m5.xlarge",
			},
			want: false,
		},
		{
			name: "returns false when machine type matches but name differs",
			operation: &internal.Operation{
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "m5.large",
							},
						},
					},
				},
			},
			additionalWorkerPool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-b",
				MachineType: "m5.large",
			},
			want: false,
		},
		{
			name: "returns false when no previous pools exist",
			operation: &internal.Operation{
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: nil,
					},
				},
			},
			additionalWorkerPool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-a",
				MachineType: "m5.large",
			},
			want: false,
		},
		{
			name: "returns true when one of multiple pools matches",
			operation: &internal.Operation{
				PreviousParameters: internal.ProvisioningParameters{
					Parameters: runtime.ProvisioningParametersDTO{
						AdditionalWorkerNodePools: []runtime.AdditionalWorkerNodePool{
							{
								Name:        "pool-a",
								MachineType: "m5.large",
							},
							{
								Name:        "pool-b",
								MachineType: "m5.xlarge",
							},
						},
					},
				},
			},
			additionalWorkerPool: runtime.AdditionalWorkerNodePool{
				Name:        "pool-b",
				MachineType: "m5.xlarge",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAdditionalWorkerPoolUnchanged(tt.operation, tt.additionalWorkerPool)
			assert.Equal(t, tt.want, got)
		})
	}
}

func newEmptyProviderSpec() *configuration.ProviderSpec {
	spec, _ := configuration.NewProviderSpec(strings.NewReader(""))
	return spec
}

func assertWorker(t *testing.T, workers []gardener.Worker, name string, zonesNumber int, zones ...string) {
	for _, worker := range workers {
		if worker.Name == name {
			assert.Len(t, worker.Zones, zonesNumber)
			assert.Subset(t, zones, worker.Zones)
			return
		}
	}
	assert.Fail(t, fmt.Sprintf("worker %s does not exists", name))
}
