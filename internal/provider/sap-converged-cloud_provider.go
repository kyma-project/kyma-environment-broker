package provider

import (
	"fmt"
	"math/rand"

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
)

type SapConvergedCloudInput struct {
	FloatingPoolName       string
	IncludeNewMachineTypes bool
}

func (p *SapConvergedCloudInput) Defaults() *gqlschema.ClusterConfigInput {
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
					Zones:                ZonesForSapConvergedCloud(DefaultSapConvergedCloudRegion),
					FloatingPoolName:     p.FloatingPoolName,
					CloudProfileName:     "converged-cloud-cp",
					LoadBalancerProvider: "f5",
				},
			},
		},
	}
}

func (p *SapConvergedCloudInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Region != nil && *pp.Parameters.Region != "" {
		input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones = ZonesForSapConvergedCloud(*pp.Parameters.Region)
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

func ZonesForSapConvergedCloud(region string) []string {
	zones, found := sapConvergedCloudZones[region]
	if !found {
		zones = "a"
	}
	zone := string(zones[rand.Intn(len(zones))])
	return []string{fmt.Sprintf("%s%s", region, zone)}
}
