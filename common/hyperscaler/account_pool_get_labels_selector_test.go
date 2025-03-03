package hyperscaler

import (
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
	"github.com/stretchr/testify/require"
)

func TestGetLabelsSelector(t *testing.T) {

	for _, platformRegion := range broker.PlatformRegions {
		for _, clusterRegion := range broker.ClusterRegions {
			for _, cloudProvider := range runtime.CloudProviders {
				for _, plan := range broker.PlanNamesMapping {

					hypType, err := HypTypeFromCloudProviderWithRegion(cloudProvider, &clusterRegion, &platformRegion)
					require.NoError(t, err)

					euAccess := euaccess.IsEURestrictedAccess(platformRegion)

					var shared = broker.IsTrialPlan(plan) || broker.IsSapConvergedCloudPlan(plan)

					labelSelector := getLabelsSelector(hypType, shared, euAccess)

					// TODO: inject new implementation
					newLabelSelector := labelSelector

					require.Equal(t, labelSelector, newLabelSelector, fmt.Sprintf("platformRegion: %s, clusterRegion: %s, cloudProvider: %s, plan: %s, labelSelector: %s, newLabelSelector: %s\n", platformRegion, clusterRegion, cloudProvider, plan, labelSelector, newLabelSelector))
				}
			}
		}
	}
}
