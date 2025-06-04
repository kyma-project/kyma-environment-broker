package provider

import (
	"fmt"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Provider interface {
	Provide() internal.ProviderValues
}

type ZonesProvider interface {
	RandomZones(cp pkg.CloudProvider, region string, zonesCount int) []string
}

type PlanConfigProvider interface {
	DefaultVolumeSizeGb(planName string) (int, bool)
}

type PlanSpecificValuesProvider struct {
	multiZoneCluster           bool
	defaultTrialProvider       pkg.CloudProvider
	useSmallerMachineTypes     bool
	trialPlatformRegionMapping map[string]string
	defaultPurpose             string
	commercialFailureTolerance string

	zonesProvider ZonesProvider
	planSpec      PlanConfigProvider
}

func NewPlanSpecificValuesProvider(cfg broker.InfrastructureManager,
	trialPlatformRegionMapping map[string]string, zonesProvider ZonesProvider, planSpec PlanConfigProvider) *PlanSpecificValuesProvider {

	return &PlanSpecificValuesProvider{
		multiZoneCluster:           cfg.MultiZoneCluster,
		defaultTrialProvider:       cfg.DefaultTrialProvider,
		useSmallerMachineTypes:     cfg.UseSmallerMachineTypes,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		defaultPurpose:             cfg.DefaultGardenerShootPurpose,
		commercialFailureTolerance: cfg.ControlPlaneFailureTolerance,
		zonesProvider:              zonesProvider,
		planSpec:                   planSpec,
	}
}

func (s *PlanSpecificValuesProvider) ValuesForPlanAndParameters(provisioningParameters internal.ProvisioningParameters) (internal.ProviderValues, error) {
	var p Provider
	switch provisioningParameters.PlanID {
	case broker.AWSPlanID, broker.BuildRuntimeAWSPlanID:
		p = &AWSInputProvider{
			Purpose:                s.defaultPurpose,
			MultiZone:              s.multiZoneCluster,
			ProvisioningParameters: provisioningParameters,
			FailureTolerance:       s.commercialFailureTolerance,
			ZonesProvider:          s.zonesProvider,
		}
	case broker.PreviewPlanID:
		p = &AWSInputProvider{
			Purpose:                s.defaultPurpose,
			MultiZone:              s.multiZoneCluster,
			ProvisioningParameters: provisioningParameters,
			FailureTolerance:       s.commercialFailureTolerance,
			ZonesProvider:          s.zonesProvider,
		}
	case broker.AzurePlanID, broker.BuildRuntimeAzurePlanID:
		p = &AzureInputProvider{
			Purpose:                s.defaultPurpose,
			MultiZone:              s.multiZoneCluster,
			ProvisioningParameters: provisioningParameters,
			FailureTolerance:       s.commercialFailureTolerance,
			ZonesProvider:          s.zonesProvider,
		}
	case broker.AzureLitePlanID:
		p = &AzureLiteInputProvider{
			Purpose:                s.defaultPurpose,
			UseSmallerMachineTypes: s.useSmallerMachineTypes,
			ProvisioningParameters: provisioningParameters,
			ZonesProvider:          s.zonesProvider,
		}
	case broker.GCPPlanID, broker.BuildRuntimeGCPPlanID:
		p = &GCPInputProvider{
			Purpose:                s.defaultPurpose,
			MultiZone:              s.multiZoneCluster,
			ProvisioningParameters: provisioningParameters,
			FailureTolerance:       s.commercialFailureTolerance,
			ZonesProvider:          s.zonesProvider,
		}
	case broker.FreemiumPlanID:
		switch provisioningParameters.PlatformProvider {
		case pkg.AWS:
			p = &AWSFreemiumInputProvider{
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: provisioningParameters,
				ZonesProvider:          s.zonesProvider,
			}
		case pkg.Azure:
			p = &AzureFreemiumInputProvider{
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: provisioningParameters,
				ZonesProvider:          s.zonesProvider,
			}
		default:
			return internal.ProviderValues{}, fmt.Errorf("freemium provider for '%s' is not supported", provisioningParameters.PlatformProvider)
		}
	case broker.SapConvergedCloudPlanID:
		p = &SapConvergedCloudInputProvider{
			Purpose:                s.defaultPurpose,
			MultiZone:              s.multiZoneCluster,
			ProvisioningParameters: provisioningParameters,
			FailureTolerance:       s.commercialFailureTolerance,
			ZonesProvider:          s.zonesProvider,
		}
	case broker.TrialPlanID:
		var trialProvider pkg.CloudProvider
		if provisioningParameters.Parameters.Provider == nil {
			trialProvider = s.defaultTrialProvider
		} else {
			trialProvider = *provisioningParameters.Parameters.Provider
		}
		switch trialProvider {
		case pkg.AWS:
			p = &AWSTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: provisioningParameters,
				ZonesProvider:          s.zonesProvider,
			}
		case pkg.GCP:
			p = &GCPTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
				ProvisioningParameters: provisioningParameters,
				ZonesProvider:          s.zonesProvider,
			}
		case pkg.Azure:
			p = &AzureTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: provisioningParameters,
				ZonesProvider:          s.zonesProvider,
			}
		default:
			return internal.ProviderValues{}, fmt.Errorf("trial provider for %s not yet implemented", trialProvider)
		}

	case broker.OwnClusterPlanID:
		p = &OwnClusterinputProvider{}
	default:
		return internal.ProviderValues{}, fmt.Errorf("plan %s not supported", provisioningParameters.PlanID)
	}

	values := p.Provide()
	volumeSize, found := s.planSpec.DefaultVolumeSizeGb(broker.PlanNamesMapping[provisioningParameters.PlanID])
	if found {
		values.VolumeSizeGb = volumeSize
	}
	return values, nil
}

func ProviderToCloudProvider(providerType string) pkg.CloudProvider {
	switch providerType {
	case "azure":
		return pkg.Azure
	case "aws":
		return pkg.AWS
	case "gcp":
		return pkg.GCP
	case "openstack":
		return pkg.SapConvergedCloud
	default:
		return pkg.UnknownProvider
	}
}
