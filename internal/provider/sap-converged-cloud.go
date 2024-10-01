package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	DefaultSapConvergedCloudRegion         = "eu-de-1"
	DefaultSapConvergedCloudMachineType    = "g_c2_m8"
	DefaultSapConvergedCloudMultiZoneCount = 3
)

type (
	SapConvergedCloudInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *SapConvergedCloudInputProvider) Provide() Values {
	region := DefaultSapConvergedCloudRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	zonesCount := 1
	if p.MultiZone {
		zonesCount = CountZonesForSapConvergedCloud(region)
	}

	zones := ZonesForSapConvergedCloud(region, zonesCount)
	return Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "openstack",
		DefaultMachineType:   DefaultSapConvergedCloudMachineType,
		Region:               region,
		Purpose:              p.Purpose,
		DiskType:             "",
	}
}
