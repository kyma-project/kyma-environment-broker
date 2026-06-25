package provider

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	DefaultGDCHRegion         = "us-west16"
	DefaultGDCHMachineType    = "n3-standard-2-gdc"
	DefaultGDCHMultiZoneCount = 3
)

type GDCHInputProvider struct {
	Purpose                string
	MultiZone              bool
	ProvisioningParameters internal.ProvisioningParameters
	FailureTolerance       string
	ZonesProvider          ZonesProvider
}

func (p *GDCHInputProvider) Provide() internal.ProviderValues {
	zonesCount := p.zonesCount()
	region := p.region()
	zones := ZonesForGDCHRegion(region, p.ZonesProvider.RandomZones(pkg.GDCH, region, zonesCount))
	return internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         GDCHProviderType,
		Region:               region,
		Purpose:              p.Purpose,
		VolumeSizeGb:         80,
		DiskType:             "Standard",
		FailureTolerance:     &p.FailureTolerance,
	}
}

func (p *GDCHInputProvider) zonesCount() int {
	if p.MultiZone {
		return DefaultGDCHMultiZoneCount
	}
	return 1
}

func (p *GDCHInputProvider) region() string {
	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		return *p.ProvisioningParameters.Parameters.Region
	}
	return DefaultGDCHRegion
}

func ZonesForGDCHRegion(region string, zones []string) []string {
	fullNames := make([]string, 0, len(zones))
	for _, zone := range zones {
		fullNames = append(fullNames, FullZoneName(GDCHProviderType, region, zone))
	}
	return fullNames
}
