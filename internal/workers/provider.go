package workers

import (
	"fmt"
	"math/rand"
	"strconv"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/regionssupportingmachine"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Provider struct {
	imConfig                 broker.InfrastructureManager
	regionsSupportingMachine regionssupportingmachine.RegionsSupportingMachine
}

func NewProvider(imConfig broker.InfrastructureManager, regionsSupportingMachine regionssupportingmachine.RegionsSupportingMachine) *Provider {
	return &Provider{
		imConfig:                 imConfig,
		regionsSupportingMachine: regionsSupportingMachine,
	}
}

func (p *Provider) CreateAdditionalWorkers(values internal.ProviderValues, currentAdditionalWorkers map[string]gardener.Worker, additionalWorkerNodePools []pkg.AdditionalWorkerNodePool,
	zones []string, planID string) ([]gardener.Worker, error) {
	additionalWorkerNodePoolsMaxUnavailable := intstr.FromInt32(int32(0))
	workers := make([]gardener.Worker, 0, len(additionalWorkerNodePools))

	for _, additionalWorkerNodePool := range additionalWorkerNodePools {
		currentAdditionalWorker, exists := currentAdditionalWorkers[additionalWorkerNodePool.Name]

		var workerZones []string
		if exists {
			workerZones = currentAdditionalWorker.Zones
		} else {
			workerZones = zones
			customAvailableZones, err := p.regionsSupportingMachine.AvailableZones(additionalWorkerNodePool.MachineType, values.Region, planID)
			if err != nil {
				return []gardener.Worker{}, fmt.Errorf("while getting available zones from regions supporting machine: %w", err)
			}
			// If custom zones are found, use them instead of the Kyma workload zones.
			if len(customAvailableZones) > 0 {
				workerZones = customAvailableZones
			}
			if !additionalWorkerNodePool.HAZones {
				rand.Shuffle(len(workerZones), func(i, j int) { workerZones[i], workerZones[j] = workerZones[j], workerZones[i] })
				workerZones = workerZones[:1]
			}
		}
		workerMaxSurge := intstr.FromInt32(int32(len(workerZones)))

		worker := gardener.Worker{
			Name: additionalWorkerNodePool.Name,
			Machine: gardener.Machine{
				Type: additionalWorkerNodePool.MachineType,
				Image: &gardener.ShootMachineImage{
					Name:    p.imConfig.MachineImage,
					Version: &p.imConfig.MachineImageVersion,
				},
			},
			Maximum:        int32(additionalWorkerNodePool.AutoScalerMax),
			Minimum:        int32(additionalWorkerNodePool.AutoScalerMin),
			MaxSurge:       &workerMaxSurge,
			MaxUnavailable: &additionalWorkerNodePoolsMaxUnavailable,
			Zones:          workerZones,
		}

		if values.ProviderType != "openstack" {
			volumeSize := strconv.Itoa(values.VolumeSizeGb)
			worker.Volume = &gardener.Volume{
				Type:       ptr.String(values.DiskType),
				VolumeSize: fmt.Sprintf("%sGi", volumeSize),
			}
		}

		workers = append(workers, worker)
	}

	return workers, nil
}
