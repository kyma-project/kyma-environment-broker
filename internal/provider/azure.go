package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
)

type (
	AzureInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
		FailureTolerance       string
	}
	AzureTrialInputProvider struct {
		PlatformRegionMapping  map[string]string
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
	}
	AzureLiteInputProvider struct {
		Purpose                string
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
	}
	AzureFreemiumInputProvider struct {
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

func (p *AzureInputProvider) Provide() internal.ProviderValues {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "azure",
		DefaultMachineType:   DefaultAzureMachineType,
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

	zones := p.zones()
	region := p.region()

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

func (p *AzureTrialInputProvider) zones() []string {
	return GenerateAzureZones(1)
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
	machineType := DefaultOldAzureTrialMachineType
	if p.UseSmallerMachineTypes {
		machineType = DefaultAzureMachineType
	}
	zones := p.zones()
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 10,
		DefaultAutoScalerMin: 2,
		ZonesCount:           1,
		Zones:                zones,
		ProviderType:         AzureProviderType,
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              p.Purpose,
		DiskType:             "StandardSSD_LRS",
		VolumeSizeGb:         80,
		FailureTolerance:     nil,
	}
}

func (p *AzureLiteInputProvider) zones() []string {
	return GenerateAzureZones(1)
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
	zones := p.zones()
	region := DefaultAzureRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                zones,
		ProviderType:         AzureProviderType,
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		DiskType:             "Standard_LRS",
		VolumeSizeGb:         50,
		FailureTolerance:     nil,
	}
}

func (p *AzureFreemiumInputProvider) zones() []string {
	return GenerateAzureZones(1)
}
