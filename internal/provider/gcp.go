package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/assuredworkloads"
)

type (
	GCPInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
	}

	GCPTrialInputProvider struct {
		Purpose                string
		PlatformRegionMapping  map[string]string
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *GCPInputProvider) Provide() Values {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := p.region()
	return Values{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "gcp",
		DefaultMachineType:   DefaultGCPMachineType,
		Region:               region,
		Purpose:              p.Purpose,
		VolumeSizeGb:         80,
		DiskType:             "pd-balanced",
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

func (p *GCPInputProvider) region() string {
	if assuredworkloads.IsKSA(p.ProvisioningParameters.PlatformRegion) {
		return DefaultGCPAssuredWorkloadsRegion
	}

	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		return *p.ProvisioningParameters.Parameters.Region
	}

	return DefaultGCPRegion
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
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
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
	if assuredworkloads.IsKSA(p.ProvisioningParameters.PlatformRegion) {
		return DefaultGCPAssuredWorkloadsRegion
	}
	if p.ProvisioningParameters.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[p.ProvisioningParameters.PlatformRegion]
		if found {
			gpcSpecific, ok := toGCPSpecific[abstractRegion]
			if ok {
				return *gpcSpecific
			}
		}
	}

	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		gpcSpecific, ok := toGCPSpecific[*p.ProvisioningParameters.Parameters.Region]
		if ok {
			return *gpcSpecific
		}
	}

	return DefaultGCPRegion
}
