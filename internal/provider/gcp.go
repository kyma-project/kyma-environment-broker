package provider

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/assuredworkloads"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

const (
	DefaultGCPRegion                 = "europe-west3"
	DefaultGCPAssuredWorkloadsRegion = "me-central2"
	DefaultGCPMachineType            = "n2-standard-2"
	DefaultGCPTrialMachineType       = "n2-standard-4"
	DefaultGCPMultiZoneCount         = 3
)

var europeGcp = "europe-west3"
var usGcp = "us-central1"
var asiaGcp = "asia-south1"

var toGCPSpecific = map[string]*string{
	string(broker.Europe): &europeGcp,
	string(broker.Us):     &usGcp,
	string(broker.Asia):   &asiaGcp,
}

type (
	GCPInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
		FailureTolerance       string
		ZonesProvider          ZonesProvider
	}

	GCPTrialInputProvider struct {
		Purpose                string
		PlatformRegionMapping  map[string]string
		ProvisioningParameters internal.ProvisioningParameters
		ZonesProvider          ZonesProvider
	}
)

func (p *GCPInputProvider) Provide() internal.ProviderValues {
	zonesCount := p.zonesCount()
	region := p.region()
	zones := ZonesForGCPRegion(region, p.ZonesProvider.RandomZones(pkg.GCP, region, zonesCount))
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         GCPProviderType,
		Region:               region,
		Purpose:              p.Purpose,
		VolumeSizeGb:         80,
		DiskType:             "pd-balanced",
		FailureTolerance:     &p.FailureTolerance,
	}
}

func (p *GCPInputProvider) zonesCount() int {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultGCPMultiZoneCount
	}
	return zonesCount
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

func (p *GCPTrialInputProvider) Provide() internal.ProviderValues {
	region := p.region()

	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                ZonesForGCPRegion(region, p.ZonesProvider.RandomZones(pkg.GCP, region, 1)),
		ProviderType:         GCPProviderType,
		DefaultMachineType:   DefaultGCPTrialMachineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
		FailureTolerance:     nil,
	}
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

func ZonesForGCPRegion(region string, zones []string) []string {
	fullNames := []string{}
	for _, zone := range zones {
		fullNames = append(fullNames, FullZoneName(GCPProviderType, region, zone))
	}

	return fullNames
}
