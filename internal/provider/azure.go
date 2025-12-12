package provider

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
)

const (
	DefaultAzureRegion              = "eastus"
	DefaultEuAccessAzureRegion      = "switzerlandnorth"
	DefaultAzureMultiZoneCount      = 3
	DefaultAzureMachineType         = "Standard_D2s_v5"
	DefaultOldAzureTrialMachineType = "Standard_D4s_v5"
)

var europeAzure = "westeurope"
var usAzure = "eastus"
var asiaAzure = "southeastasia"

var trialPurpose = "evaluation"

var toAzureSpecific = map[string]*string{
	string(broker.Europe): &europeAzure,
	string(broker.Us):     &usAzure,
	string(broker.Asia):   &asiaAzure,
}

type (
	AzureInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
		FailureTolerance       string
		ZonesProvider          ZonesProvider
	}
	AzureTrialInputProvider struct {
		PlatformRegionMapping  map[string]string
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
		ZonesProvider          ZonesProvider
	}
	AzureLiteInputProvider struct {
		Purpose                string
		ProvisioningParameters internal.ProvisioningParameters
		ZonesProvider          ZonesProvider
	}
	AzureFreemiumInputProvider struct {
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
		ZonesProvider          ZonesProvider
	}
)

func (p *AzureInputProvider) Provide() internal.ProviderValues {
	zonesCount := p.zonesCount()
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	zones := p.ZonesProvider.RandomZones(pkg.Azure, region, zonesCount)
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "azure",
		Region:               region,
		Purpose:              p.Purpose,
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
		FailureTolerance:     &p.FailureTolerance,
	}
}

func (p *AzureInputProvider) zonesCount() int {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAzureMultiZoneCount
	}
	return zonesCount
}

func (p *AzureInputProvider) zones() []string {
	return p.generateRandomAzureZones(p.zonesCount())
}

func (p *AzureInputProvider) generateRandomAzureZones(zonesCount int) []string {
	return GenerateAzureZones(zonesCount)
}

func (p *AzureTrialInputProvider) Provide() internal.ProviderValues {
	machineType := DefaultOldAzureTrialMachineType
	if p.UseSmallerMachineTypes {
		machineType = DefaultAzureMachineType
	}

	region := p.region()
	zones := p.ZonesProvider.RandomZones(pkg.Azure, region, 1)

	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                zones,
		ProviderType:         "azure",
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
		FailureTolerance:     nil,
	}
}

func (p *AzureTrialInputProvider) region() string {
	if euaccess.IsEURestrictedAccess(p.ProvisioningParameters.PlatformRegion) {
		return DefaultEuAccessAzureRegion
	}
	if p.ProvisioningParameters.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[p.ProvisioningParameters.PlatformRegion]
		if found {
			return *toAzureSpecific[abstractRegion]
		}
	}
	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		return *toAzureSpecific[*p.ProvisioningParameters.Parameters.Region]
	}
	return DefaultAzureRegion
}

func (p *AzureLiteInputProvider) Provide() internal.ProviderValues {
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	zones := p.ZonesProvider.RandomZones(pkg.Azure, region, 1)
	zoneFullNames := []string{}
	for _, z := range zones {
		zoneFullNames = append(zoneFullNames, FullZoneName(AzureProviderType, region, z))
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 2,
		ZonesCount:           1,
		Zones:                zoneFullNames,
		ProviderType:         AzureProviderType,
		Region:               region,
		Purpose:              p.Purpose,
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
		FailureTolerance:     nil,
	}
}

func (p *AzureLiteInputProvider) region() string {
	if euaccess.IsEURestrictedAccess(p.ProvisioningParameters.PlatformRegion) {
		return DefaultEuAccessAzureRegion
	}
	return DefaultAzureRegion
}

func (p *AzureFreemiumInputProvider) Provide() internal.ProviderValues {
	machineType := DefaultOldAzureTrialMachineType
	if p.UseSmallerMachineTypes {
		machineType = DefaultAzureMachineType
	}
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	zones := p.ZonesProvider.RandomZones(pkg.Azure, region, 1)
	zoneFullNames := []string{}
	for _, z := range zones {
		zoneFullNames = append(zoneFullNames, FullZoneName(AzureProviderType, region, z))
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                zoneFullNames,
		ProviderType:         AzureProviderType,
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
		FailureTolerance:     nil,
	}
}
