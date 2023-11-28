package azure

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
)

func mapRegion(credentials hyperscaler.Credentials, parameters internal.ProvisioningParameters) (string, error) {
	if credentials.HyperscalerType.GetKey() != hyperscaler.Azure().GetKey() {
		return "", fmt.Errorf("cannot use credential for hyperscaler of type %v on hyperscaler of type %v", credentials.HyperscalerType.GetKey(), hyperscaler.Azure().GetKey())
	}
	if parameters.Parameters.Region == nil || *(parameters.Parameters.Region) == "" {
		return provider.DefaultAzureRegion, nil
	}
	region := *(parameters.Parameters.Region)
	switch parameters.PlanID {
	case broker.AzurePlanID, broker.AzureLitePlanID:
		if !isInList(broker.AzureRegions(euaccess.IsEURestrictedAccess(parameters.PlatformRegion)), region) {
			return "", fmt.Errorf("supplied region \"%v\" is not a valid region for Azure", region)
		}

	case broker.GCPPlanID:
		if azureRegion, mappingExists := gcpToAzureRegionMapping()[region]; mappingExists {
			region = azureRegion
			break
		}
		return "", fmt.Errorf("supplied gcp region \"%v\" cannot be mapped to Azure", region)
	default:
		return "", fmt.Errorf("cannot map from PlanID %v to azure regions", parameters.PlanID)
	}
	return region, nil
}

func isInList(list []string, item string) bool {
	for _, val := range list {
		if val == item {
			return true
		}
	}
	return false
}
