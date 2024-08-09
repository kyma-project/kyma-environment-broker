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
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *SapConvergedCloudInputProvider) Provide() Values {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := DefaultSapConvergedCloudRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "openstack",
		DefaultMachineType:   DefaultSapConvergedCloudMachineType,
		Region:               region,
		Purpose:              PurposeProduction,
		DiskType:             "",
	}
}

func (p *SapConvergedCloudInputProvider) zonesCount() int {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultSapConvergedCloudMultiZoneCount
	}
	return zonesCount
}

func (p *SapConvergedCloudInputProvider) zones() []string {
	region := DefaultSapConvergedCloudRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return ZonesForSapConvergedCloud(region, p.zonesCount())
}
