package provider

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
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
		FailureTolerance       string
		ZonesProvider          ZonesProvider
	}
)

func (p *SapConvergedCloudInputProvider) Provide() internal.ProviderValues {
	region := DefaultSapConvergedCloudRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	zonesCount := 1
	if p.MultiZone {
		zonesCount = 3
	}

	zones := ZonesForSapConvergedCloud(region, p.ZonesProvider.RandomZones(pkg.SapConvergedCloud, region, zonesCount))
	if len(zones) < zonesCount {
		zonesCount = len(zones)
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         OpenstackProviderType,
		Region:               region,
		Purpose:              p.Purpose,
		DiskType:             "",
		FailureTolerance:     &p.FailureTolerance,
	}
}
