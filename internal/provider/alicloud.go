package provider

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	DefaultAlicloudRegion         = "eu-central-1"
	DefaultAlicloudMachineType    = "ecs.g8i.large"
	DefaultAlicloudMultiZoneCount = 3
	DefaultDiskType               = "cloud_essd"
)

type (
	AlicloudInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
		FailureTolerance       string
		ZonesProvider          ZonesProvider
	}
)

func (p *AlicloudInputProvider) Provide() internal.ProviderValues {
	region := DefaultAlicloudRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAlicloudMultiZoneCount
	}

	zones := p.ZonesProvider.RandomZones(pkg.Alicloud, region, zonesCount)
	if len(zones) < zonesCount {
		zonesCount = len(zones)
	}
	for i, zone := range zones {
		zones[i] = FullZoneName("alicloud", region, zone)
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "alicloud",
		DefaultMachineType:   DefaultAlicloudMachineType,
		Region:               region,
		Purpose:              p.Purpose,
		VolumeSizeGb:         80,
		DiskType:             DefaultDiskType,
		FailureTolerance:     &p.FailureTolerance,
	}
}
