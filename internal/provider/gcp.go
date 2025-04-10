package provider

import (
	"fmt"
	"math/rand"

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
	}

	GCPTrialInputProvider struct {
		Purpose                string
		PlatformRegionMapping  map[string]string
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *GCPInputProvider) Provide() internal.ProviderValues {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := p.region()
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         GCPProviderType,
		DefaultMachineType:   DefaultGCPMachineType,
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

func (p *GCPTrialInputProvider) Provide() internal.ProviderValues {
	region := p.region()

	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                ZonesForGCPRegion(region, 1),
		ProviderType:         GCPProviderType,
		DefaultMachineType:   DefaultGCPTrialMachineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		VolumeSizeGb:         30,
		DiskType:             "pd-standard",
		FailureTolerance:     nil,
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

func ZonesForGCPRegion(region string, zonesCount int) []string {
	availableZones := []string{"a", "b", "c"}
	var zones []string
	if zonesCount > len(availableZones) {
		zonesCount = len(availableZones)
	}

	availableZones = availableZones[:zonesCount]

	rand.Shuffle(zonesCount, func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })

	for i := 0; i < zonesCount; i++ {
		zones = append(zones, fmt.Sprintf("%s-%s", region, availableZones[i]))
	}

	return zones
}
