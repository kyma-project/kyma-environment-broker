package provider

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	DefaultSapConvergedCloudRegion         = "eu-de-1"
	DefaultExposureClass                   = "converged-cloud-internet"
	DefaultSapConvergedCloudMachineType    = "g_c2_m8"
	DefaultOldSapConvergedCloudMachineType = "g_c4_m16"
	DefaultSapConvergedCloudMultiZoneCount = 3
)

type SapConvergedCloudInput struct {
	MultiZone              bool
	FloatingPoolName       string
	IncludeNewMachineTypes bool
}

func (p *SapConvergedCloudInput) Defaults() *gqlschema.ClusterConfigInput {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultSapConvergedCloudMultiZoneCount
	}
	machineType := DefaultOldSapConvergedCloudMachineType
	if p.IncludeNewMachineTypes {
		machineType = DefaultSapConvergedCloudMachineType
	}
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:          nil,
			MachineType:       machineType,
			Region:            DefaultSapConvergedCloudRegion,
			Provider:          "openstack",
			WorkerCidr:        networking.DefaultNodesCIDR,
			AutoScalerMin:     3,
			AutoScalerMax:     20,
			MaxSurge:          1,
			MaxUnavailable:    0,
			ExposureClassName: ptr.String(DefaultExposureClass),
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				OpenStackConfig: &gqlschema.OpenStackProviderConfigInput{
					Zones:                ZonesForSapConvergedCloud(DefaultSapConvergedCloudRegion, zonesCount),
					FloatingPoolName:     p.FloatingPoolName,
					CloudProfileName:     "converged-cloud-cp",
					LoadBalancerProvider: "f5",
				},
			},
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
