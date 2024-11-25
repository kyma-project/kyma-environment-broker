package provider

import (
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
)

type SapConvergedCloudInput struct {
	MultiZone                    bool
	ControlPlaneFailureTolerance string
}

func (p *SapConvergedCloudInput) Defaults() *gqlschema.ClusterConfigInput {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultSapConvergedCloudMultiZoneCount
	}

	var controlPlaneFailureTolerance *string = nil
	if p.ControlPlaneFailureTolerance != "" {
		controlPlaneFailureTolerance = &p.ControlPlaneFailureTolerance
	}
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			Provider:       "openstack",
			Region:         DefaultSapConvergedCloudRegion,
			MachineType:    DefaultSapConvergedCloudMachineType,
			DiskType:       nil,
			WorkerCidr:     networking.DefaultNodesCIDR,
			AutoScalerMin:  3,
			AutoScalerMax:  20,
			MaxSurge:       zonesCount,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				OpenStackConfig: &gqlschema.OpenStackProviderConfigInput{
					Zones:                ZonesForSapConvergedCloud(DefaultSapConvergedCloudRegion, zonesCount),
					LoadBalancerProvider: "f5",
				},
			},
			ControlPlaneFailureTolerance: controlPlaneFailureTolerance,
		},
	}
}

func (p *SapConvergedCloudInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultSapConvergedCloudMultiZoneCount
	}
	if pp.Parameters.Region != nil && *pp.Parameters.Region != "" {
		input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones = ZonesForSapConvergedCloud(*pp.Parameters.Region, zonesCount)
	}
}

func (p *SapConvergedCloudInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *SapConvergedCloudInput) Provider() pkg.CloudProvider {
	return pkg.SapConvergedCloud
}
