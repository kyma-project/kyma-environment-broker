package provider

import (
	"fmt"
	"math/rand"

	"github.com/kyma-project/kyma-environment-broker/internal/assuredworkloads"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
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
	GcpInput struct {
		MultiZone                    bool
		ControlPlaneFailureTolerance string
	}
	GcpTrialInput struct {
		PlatformRegionMapping map[string]string
	}
)

func (p *GcpInput) Defaults() *gqlschema.ClusterConfigInput {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultGCPMultiZoneCount
	}
	var controlPlaneFailureTolerance *string = nil
	if p.ControlPlaneFailureTolerance != "" {
		controlPlaneFailureTolerance = &p.ControlPlaneFailureTolerance
	}
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("pd-balanced"),
			VolumeSizeGb:   ptr.Integer(80),
			MachineType:    DefaultGCPMachineType,
			Region:         DefaultGCPRegion,
			Provider:       "gcp",
			WorkerCidr:     networking.DefaultNodesCIDR,
			AutoScalerMin:  3,
			AutoScalerMax:  20,
			MaxSurge:       zonesCount,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{
					Zones: ZonesForGCPRegion(DefaultGCPRegion, zonesCount),
				},
			},
			ControlPlaneFailureTolerance: controlPlaneFailureTolerance,
		},
	}
}

func (p *GcpInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	if pp.Parameters.Networking != nil {
		input.GardenerConfig.WorkerCidr = pp.Parameters.Networking.NodesCidr
	}
	switch {
	// explicit zones list is provided
	case len(pp.Parameters.Zones) > 0:
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, pp.Parameters.Zones)
	// region is provided, with or without zonesCount
	case pp.Parameters.Region != nil && *pp.Parameters.Region != "":
		zonesCount := 1
		if p.MultiZone {
			zonesCount = DefaultGCPMultiZoneCount
		}
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, ZonesForGCPRegion(*pp.Parameters.Region, zonesCount))
	case assuredworkloads.IsKSA(pp.PlatformRegion):
		input.GardenerConfig.Region = DefaultGCPAssuredWorkloadsRegion
		zonesCount := 1
		if p.MultiZone {
			zonesCount = DefaultGCPMultiZoneCount
		}
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, ZonesForGCPRegion(DefaultGCPAssuredWorkloadsRegion, zonesCount))
	}
}

func (p *GcpInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileProduction
}

func (p *GcpInput) Provider() pkg.CloudProvider {
	return pkg.GCP
}

func (p *GcpTrialInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			DiskType:       ptr.String("pd-standard"),
			VolumeSizeGb:   ptr.Integer(30),
			MachineType:    DefaultGCPTrialMachineType,
			Region:         DefaultGCPRegion,
			Provider:       "gcp",
			WorkerCidr:     "10.250.0.0/19",
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			MaxSurge:       1,
			MaxUnavailable: 0,
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{
					Zones: ZonesForGCPRegion(DefaultGCPRegion, 1),
				},
			},
		},
	}
}

func (p *GcpTrialInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	params := pp.Parameters
	var region string

	// if there is a platform region - use it
	if pp.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[pp.PlatformRegion]
		if found {
			region = *toGCPSpecific[abstractRegion]
		}
	}

	// if the user provides a region - use this one
	if params.Region != nil && *params.Region != "" {
		region = *toGCPSpecific[*params.Region]
	}

	if assuredworkloads.IsKSA(pp.PlatformRegion) {
		region = DefaultGCPAssuredWorkloadsRegion
	}

	// region is not empty - it means override the default one
	if region != "" {
		updateString(&input.GardenerConfig.Region, &region)
		updateSlice(&input.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones, ZonesForGCPRegion(region, 1))
	}
}

func (p *GcpTrialInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *GcpTrialInput) Provider() pkg.CloudProvider {
	return pkg.GCP
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
