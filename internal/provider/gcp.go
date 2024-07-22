package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type (
	GCPInputProvider struct {
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}

	GCPTrialInputProvider struct {
		PlatformRegionMapping  map[string]string
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

func (p *GCPTrialInputProvider) Provide() Values {
	region := p.region()

	return Values{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                ZonesForGCPRegion(region, 1),
		ProviderType:         "gcp",
		DefaultMachineType:   DefaultGCPTrialMachineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
	}
}

func (p *GCPTrialInputProvider) zones() []string {
	region := DefaultGCPRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return ZonesForGCPRegion(region, 1)
}

func (p *GCPTrialInputProvider) region() string {
	if p.ProvisioningParameters.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[p.ProvisioningParameters.PlatformRegion]
		if found {
			return *toGCPSpecific[abstractRegion]
		}
	}

	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		return *toGCPSpecific[*p.ProvisioningParameters.Parameters.Region]
	}

	return DefaultGCPRegion
}
