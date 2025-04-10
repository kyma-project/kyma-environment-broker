package provider

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
)

const (
	DefaultAWSRegion              = "eu-central-1"
	DefaultAWSTrialRegion         = "eu-west-1"
	DefaultEuAccessAWSRegion      = "eu-central-1"
	DefaultAWSMultiZoneCount      = 3
	DefaultAWSMachineType         = "m6i.large"
	DefaultOldAWSTrialMachineType = "m5.xlarge"
)

var europeAWS = "eu-west-1"
var usAWS = "us-east-1"
var asiaAWS = "ap-southeast-1"

var toAWSSpecific = map[string]*string{
	string(broker.Europe): &europeAWS,
	string(broker.Us):     &usAWS,
	string(broker.Asia):   &asiaAWS,
}

type (
	AWSInputProvider struct {
		Purpose                string
		MultiZone              bool
		ProvisioningParameters internal.ProvisioningParameters
		FailureTolerance       string
	}
	AWSTrialInputProvider struct {
		PlatformRegionMapping  map[string]string
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
	}
	AWSFreemiumInputProvider struct {
		UseSmallerMachineTypes bool
		ProvisioningParameters internal.ProvisioningParameters
	}
)

// awsZones defines a possible suffixes for given AWS regions
// The table is tested in a unit test to check if all necessary regions are covered
var awsZones = map[string]string{
	"eu-central-1":   "abc",
	"eu-west-2":      "abc",
	"ca-central-1":   "abd",
	"sa-east-1":      "abc",
	"us-east-1":      "abcdf",
	"us-west-2":      "abcd",
	"ap-northeast-1": "acd",
	"ap-northeast-2": "abc",
	"ap-south-1":     "abc",
	"ap-southeast-1": "abc",
	"ap-southeast-2": "abc",
}

func (p *AWSInputProvider) Provide() internal.ProviderValues {
	zonesCount := p.zonesCount()
	zones := p.zones()
	region := DefaultAWSRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return internal.ProviderValues{
		DefaultAutoScalerMax: 20,
		DefaultAutoScalerMin: 3,
		ZonesCount:           zonesCount,
		Zones:                zones,
		ProviderType:         "aws",
		DefaultMachineType:   DefaultAWSMachineType,
		Region:               region,
		Purpose:              p.Purpose,
		VolumeSizeGb:         80,
		DiskType:             "gp3",
		FailureTolerance:     &p.FailureTolerance,
	}
}

func (p *AWSInputProvider) zonesCount() int {
	zonesCount := 1
	if p.MultiZone {
		zonesCount = DefaultAWSMultiZoneCount
	}
	return zonesCount
}

func (p *AWSInputProvider) zones() []string {
	region := DefaultAWSRegion
	if p.ProvisioningParameters.Parameters.Region != nil {
		region = *p.ProvisioningParameters.Parameters.Region
	}
	return MultipleZonesForAWSRegion(region, p.zonesCount())
}

func (p *AWSTrialInputProvider) Provide() internal.ProviderValues {
	machineType := DefaultOldAWSTrialMachineType
	if p.UseSmallerMachineTypes {
		machineType = DefaultAWSMachineType
	}
	region := p.region()

	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                MultipleZonesForAWSRegion(region, 1),
		ProviderType:         "aws",
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		VolumeSizeGb:         50,
		DiskType:             "gp3",
		FailureTolerance:     nil,
	}
}

func (p *AWSTrialInputProvider) region() string {
	if euaccess.IsEURestrictedAccess(p.ProvisioningParameters.PlatformRegion) {
		return DefaultEuAccessAWSRegion
	}
	if p.ProvisioningParameters.PlatformRegion != "" {
		abstractRegion, found := p.PlatformRegionMapping[p.ProvisioningParameters.PlatformRegion]
		if found {
			awsSpecific, ok := toAWSSpecific[abstractRegion]
			if ok {
				return *awsSpecific
			}
		}
	}
	if p.ProvisioningParameters.Parameters.Region != nil && *p.ProvisioningParameters.Parameters.Region != "" {
		awsSpecific, ok := toAWSSpecific[*p.ProvisioningParameters.Parameters.Region]
		if ok {
			return *awsSpecific
		}
	}
	return DefaultAWSTrialRegion
}

func (p *AWSFreemiumInputProvider) Provide() internal.ProviderValues {
	machineType := DefaultOldAWSTrialMachineType
	if p.UseSmallerMachineTypes {
		machineType = DefaultAWSMachineType
	}
	region := p.region()

	return internal.ProviderValues{
		DefaultAutoScalerMax: 1,
		DefaultAutoScalerMin: 1,
		ZonesCount:           1,
		Zones:                MultipleZonesForAWSRegion(region, 1),
		ProviderType:         AWSProviderType,
		DefaultMachineType:   machineType,
		Region:               region,
		Purpose:              PurposeEvaluation,
		VolumeSizeGb:         50,
		DiskType:             "gp3",
		FailureTolerance:     nil,
	}
}

func (p *AWSFreemiumInputProvider) region() string {
	if euaccess.IsEURestrictedAccess(p.ProvisioningParameters.PlatformRegion) {
		return DefaultEuAccessAWSRegion
	}
	return DefaultAWSRegion
}
