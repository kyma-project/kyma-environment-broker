package workers

import (
	"fmt"
	"log/slog"
	"strconv"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Provider struct {
	log          *slog.Logger
	imConfig     broker.InfrastructureManager
	providerSpec *configuration.ProviderSpec
}

func NewProvider(log *slog.Logger, imConfig broker.InfrastructureManager, providerSpec *configuration.ProviderSpec) *Provider {
	return &Provider{
		log:          log,
		imConfig:     imConfig,
		providerSpec: providerSpec,
	}
}

func (p *Provider) CreateAdditionalWorkers(values internal.ProviderValues, currentAdditionalWorkers map[string]gardener.Worker, additionalWorkerNodePools []pkg.AdditionalWorkerNodePool,
	zones []string, planID string, discoveredZones map[string][]string) ([]gardener.Worker, error) {
	additionalWorkerNodePoolsMaxUnavailable := intstr.FromInt32(int32(0))
	workers := make([]gardener.Worker, 0, len(additionalWorkerNodePools))

	for _, additionalWorkerNodePool := range additionalWorkerNodePools {
		currentAdditionalWorker, exists := currentAdditionalWorkers[additionalWorkerNodePool.Name]

		var workerZones []string
		if exists {
			workerZones = currentAdditionalWorker.Zones
		} else {
			workerZones = zones

			if p.providerSpec.ZonesDiscovery(pkg.CloudProviderFromString(values.ProviderType)) {
				// If zones discovery is enabled, use zones resolved at runtime
				workerZones = discoveredZones[additionalWorkerNodePool.MachineType]
			} else {
				customAvailableZones, err := p.providerSpec.AvailableZonesForAdditionalWorkers(additionalWorkerNodePool.MachineType, values.Region, values.ProviderType)
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
			}

			// limit to 3 zones (if there is more than 3 available)
			if len(workerZones) > 3 {
				workerZones = workerZones[:3]
			}
			if !additionalWorkerNodePool.HAZones || planID == broker.AzureLitePlanID {
				workerZones = workerZones[:1]
			}
		}
		p.log.Info(fmt.Sprintf("Zones for %s additional worker node pool: %v", additionalWorkerNodePool.Name, workerZones))
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
