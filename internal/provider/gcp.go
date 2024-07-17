package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type (
	GCPInputProvider struct {
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *GCPInputProvider) Provide() Values {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := DefaultGCPRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "gcp",
		DefaultMachineType:   DefaultGCPMachineType,
		Region:               region,
		Purpose:              PurposeProduction,
	}
}

func (p *GCPInputProvider) zonesCount() int {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultGCPMultiZoneCount
	}
	return zonesCount
}

func (p *GCPInputProvider) zones() []string {
	region := DefaultGCPRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return ZonesForGCPRegion(region, p.zonesCount())
}
