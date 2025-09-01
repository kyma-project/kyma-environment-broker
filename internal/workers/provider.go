package workers

import (
	"fmt"
	"strconv"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type RegionsSupportingMachine interface {
	AvailableZonesForAdditionalWorkers(machineType, region, providerType string) ([]string, error)
}

type Provider struct {
	imConfig broker.InfrastructureManager

	regionsSupportingMachine RegionsSupportingMachine
}

func NewProvider(imConfig broker.InfrastructureManager, regionsSupportingMachine RegionsSupportingMachine) *Provider {
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
			customAvailableZones, err := p.regionsSupportingMachine.AvailableZonesForAdditionalWorkers(additionalWorkerNodePool.MachineType, values.Region, values.ProviderType)
			if err != nil {
				return []gardener.Worker{}, fmt.Errorf("while getting available zones from regions supporting machine: %w", err)
			}

			// If custom zones are found, use them instead of the Kyma workload zones.
			if len(customAvailableZones) > 0 {
				var formattedZones []string
				for _, zone := range customAvailableZones {
					formattedZones = append(formattedZones, provider.FullZoneName(values.ProviderType, values.Region, zone))
				}
				workerZones = formattedZones
			}
			// limit to 3 zones (if there is more than 3 available)
			if len(workerZones) > 3 {
				workerZones = workerZones[:3]
			}
			if !additionalWorkerNodePool.HAZones || planID == broker.AzureLitePlanID {
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

		// Add hardcoded labels, annotations, and taints to the worker pool
		worker.Labels = map[string]string{
			"kyma.project.io/worker-pool": additionalWorkerNodePool.Name,
			"environment":                 "production",
			"team":                        "kyma",
		}

		worker.Annotations = map[string]string{
			"kyma.project.io/created-by": "kyma-environment-broker",
			"cluster.x-k8s.io/machine":   "managed",
			"description":                "Additional worker node pool managed by KEB",
		}

		worker.Taints = []corev1.Taint{
			{
				Key:    "workload-type",
				Value:  "additional-worker",
				Effect: corev1.TaintEffectNoSchedule,
			},
			{
				Key:    "managed-by",
				Value:  "keb",
				Effect: corev1.TaintEffectPreferNoSchedule,
			},
		}

		workers = append(workers, worker)
	}

	return workers, nil
}
