package provider

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Provider interface {
	Provide() Values
}

func GenerateValues(
	operation *internal.Operation,
	multiZoneCluster bool,
	defaultTrialProvider internal.CloudProvider,
	useSmallerMachineTypes bool,
	trialPlatformRegionMapping map[string]string,
	defaultPurpose string,
) (Values, error) {
	var p Provider
	switch operation.ProvisioningParameters.PlanID {
	case broker.AWSPlanID:
		p = &AWSInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.PreviewPlanID:
		p = &AWSInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.AzurePlanID:
		p = &AzureInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.AzureLitePlanID:
		p = &AzureLiteInputProvider{
			Purpose:                defaultPurpose,
			UseSmallerMachineTypes: useSmallerMachineTypes,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.GCPPlanID:
		p = &GCPInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.FreemiumPlanID:
		switch operation.ProvisioningParameters.PlatformProvider {
		case internal.AWS:
			p = &AWSFreemiumInputProvider{
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.Azure:
			p = &AzureFreemiumInputProvider{
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		default:
			return Values{}, fmt.Errorf("freemium provider for '%s' is not supported", operation.ProvisioningParameters.PlatformProvider)
		}
	case broker.SapConvergedCloudPlanID:
		p = &SapConvergedCloudInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.TrialPlanID:
		var trialProvider internal.CloudProvider
		if operation.ProvisioningParameters.Parameters.Provider == nil {
			trialProvider = defaultTrialProvider
		} else {
			trialProvider = *operation.ProvisioningParameters.Parameters.Provider
		}
		switch trialProvider {
		case internal.AWS:
			p = &AWSTrialInputProvider{
				PlatformRegionMapping:  trialPlatformRegionMapping,
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.GCP:
			p = &GCPTrialInputProvider{
				PlatformRegionMapping:  trialPlatformRegionMapping,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.Azure:
			p = &AzureTrialInputProvider{
				PlatformRegionMapping:  trialPlatformRegionMapping,
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		default:
			return Values{}, fmt.Errorf("trial provider for %s not yet implemented", trialProvider)
		}

	default:
		return Values{}, fmt.Errorf("plan %s not supported", operation.ProvisioningParameters.PlanID)
	}
	return p.Provide(), nil
}
