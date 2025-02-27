package hyperscaler

import (
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/stretchr/testify/require"
)

func TestGetLabelsSelector(t *testing.T) {

	var platformRegions = []string{"cf-ap21",
		"cf-us20", "cf-jp20", "cf-us21", "cf-eu20", "cf-ap20", "cf-br20", "cf-ca20", "cf-ch20", "cf-us10", "cf-eu10", "cf-eu11", "cf-br10", "cf-jp10", "cf-ca10", "cf-ap12", "cf-ap10", "cf-ap11", "cf-us11", "cf-us30", "cf-eu30", "cf-in30", "cf-jp30", "cf-jp31", "cf-sa30", "cf-sa31", "cf-il30", "cf-br30", "cf-ap30",
	}

	var clusterRegions = []string{
		"centralus",
		"eastus",
		"westus2",
		"northeurope",
		"uksouth",
		"japaneast",
		"southeastasia",
		"westeurope",
		"australiaeast",
		"switzerlandnorth",
		"brazilsouth",
		"canadacentral",
		"eu-central-1",
		"eu-west-2",
		"ca-central-1",
		"sa-east-1",
		"us-east-1",
		"us-west-1",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-south-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"us-west-2",
		"eu-central-1",
		"us-east-1",
		"ap-southeast-1",
		"europe-west3",
		"us-central1",
		"asia-south1",
		"asia-northeast2",
		"me-central2",
		"me-west1",
		"australia-southeast1",
		"southamerica-east1",
		"asia-northeast1",
		"asia-southeast1",
		"us-west1",
		"us-east4",
		"europe-west4",
	}
	cloudProviders := []pkg.CloudProvider{runtime.Azure, runtime.AWS, runtime.GCP, runtime.SapConvergedCloud}

	for _, platformRegion := range platformRegions {
		for _, clusterRegion := range clusterRegions {
			for _, cloudProvider := range cloudProviders {
                for _, plan := range broker.PlanNamesMapping {
                    effectiveRegion := provider.GetEffectiveRegionForSapConvergedCloud(&clusterRegion)

                    hypType, err := HypTypeFromCloudProviderWithRegion(cloudProvider, &effectiveRegion, &platformRegion)
                    require.NoError(t, err)

                    euAccess := euaccess.IsEURestrictedAccess(platformRegion)

                    var shared = broker.IsTrialPlan(plan) || broker.IsSapConvergedCloudPlan(plan)

                    labelSelector := getLabelsSelector(hypType, shared, euAccess)

					// TODO: inject new implementation
					newLabelSelector := labelSelector

					require.Equal(t, labelSelector, newLabelSelector, 	fmt.Sprintf("platformRegion: %s, clusterRegion: %s, cloudProvider: %s, plan: %s, labelSelector: %s, newLabelSelector: %s\n", platformRegion, clusterRegion, cloudProvider, plan, labelSelector, newLabelSelector))
                }
			}
		}
	}
}
