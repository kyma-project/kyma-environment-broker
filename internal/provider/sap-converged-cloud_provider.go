package provider

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
)

const (
	DefaultSapConvergedCloudRegion         = "eu-de-1"
	DefaultSapConvergedCloudMachineType    = "g_c2_m8"
	DefaultSapConvergedCloudMultiZoneCount = 3
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

func (p *SapConvergedCloudInput) Provider() internal.CloudProvider {
	return internal.SapConvergedCloud
}

// sapConvergedCloudZones defines a possible suffixes for given OpenStack regions
// The table is tested in a unit test to check if all necessary regions are covered
var sapConvergedCloudZones = map[string]string{
	"eu-de-1": "abd",
	"ap-au-1": "ab",
	"na-us-1": "abd",
	"eu-de-2": "ab",
	"na-us-2": "ab",
	"ap-jp-1": "a",
	"ap-ae-1": "ab",
}

func ZonesForSapConvergedCloud(region string, zonesCount int) []string {
	zones, found := sapConvergedCloudZones[region]
	if !found {
		zones = "a"
		zonesCount = 1
	}

	availableZones := strings.Split(zones, "")
	rand.Shuffle(len(availableZones), func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })
	if zonesCount > len(availableZones) {
		// get maximum number of zones for region
		zonesCount = len(availableZones)
	}

	availableZones = availableZones[:zonesCount]

	var generatedZones []string
	for _, zone := range availableZones {
		generatedZones = append(generatedZones, fmt.Sprintf("%s%s", region, zone))
	}
	return generatedZones
}
