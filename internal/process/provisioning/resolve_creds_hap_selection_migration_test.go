package provisioning

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"

	k8sTesting "k8s.io/client-go/testing"
)

// To generate output hap-old-implementation files (used to compare results by hand) switch writeFiles to true
func TestResolveCredentials_IntegrationAzure2(t *testing.T) {
	writeFiles := false

	memoryStorage := storage.NewMemoryStorage()

	unstructuredList := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{},
	}

	var oldImplementationLog *bufio.Writer
	var newImplementationLog *bufio.Writer

	if writeFiles {
		oldLog, err := os.OpenFile("./hap-old-implementation.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModeAppend)
		require.NoError(t, err)
		defer oldLog.Close()

		oldImplementationLog = bufio.NewWriter(oldLog)

		newLog, err := os.OpenFile("./hap-new-implementation.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModeAppend)
		require.NoError(t, err)
		defer newLog.Close()

		newImplementationLog = bufio.NewWriter(newLog)
	}

	gc := gardener.NewDynamicFakeClient()
	gc.PrependReactor("get", gardener.SecretBindingResource.Resource, func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, fixDummySecretBinding(""), nil
	})

	// in memory storage for queried labels and results, used for
	// asserting if the same labels do not produce different secrets queries
	labelsResultsMap := map[string]*unstructured.Unstructured{}

	// used to compare labels between two runs
	// key - string - is originally queried label
	// value - int - is used to check how many times a secret has been queried
	queriesCountMap := map[string]int{}
	increase := true

	gc.PrependReactor("list", gardener.SecretBindingResource.Resource, func(action k8sTesting.Action) (bool, runtime.Object, error) {
		originalLabel := action.(k8sTesting.ListActionImpl).GetListRestrictions().Labels.String()

		if increase {
			if writeFiles {
				n, err := oldImplementationLog.WriteString(originalLabel + "\n")
				require.NoError(t, err)
				require.NotZero(t, n)
			}
		} else {
			if writeFiles {
				n, err := newImplementationLog.WriteString(originalLabel + "\n")
				require.NoError(t, err)
				require.NotZero(t, n)
			}
		}

		// count query times
		if _, ok := queriesCountMap[originalLabel]; !ok {
			queriesCountMap[originalLabel] = 1
		} else {
			if increase {
				queriesCountMap[originalLabel]++
			} else {
				queriesCountMap[originalLabel]--
			}
		}

		if _, ok := labelsResultsMap[originalLabel]; !ok {

			// filter out labels that select secrets without that label
			// for example !dirty selects a secret without dirty label
			filteredSplit := []string{}
			split := strings.Split(originalLabel, ",")
			for _, label := range split {
				label = strings.Trim(label, " ")
				if label != "" && label != " " && label != "!dirty" && label != "!euAccess" && label != "shared!=true" && label != "!tenantName" {
					// at this point all labels should be in format key=value
					require.Contains(t, label, "=", "Found a single value label: %s", label)
					filteredSplit = append(filteredSplit, label)
				}
			}

			// secret is queried for the first time - new label, add
			finalLabels := strings.Join(filteredSplit, ",")
			labelsResultsMap[originalLabel] = fixDummySecretBinding(finalLabels)

			// add secret so that invoked step can continue its execution
			unstructuredList.Items = append(unstructuredList.Items, *labelsResultsMap[originalLabel])
		}

		return true, unstructuredList, nil
	})

	providers := []pkg.CloudProvider{pkg.AWS,
		pkg.Azure, pkg.GCP, pkg.SapConvergedCloud,
	}

	accountProvider := hyperscaler.NewAccountProvider(hyperscaler.NewAccountPool(gc, namespace), hyperscaler.NewSharedGardenerAccountPool(gc, namespace))

	for _, planId := range broker.PlanIDsMapping {
		for _, platformRegion := range platformRegions {
			for _, clusterRegion := range clusterRegions {
				for _, provider := range providers {

					if writeFiles {
						n, err := oldImplementationLog.WriteString(fmt.Sprintf("INPUT: PlanID: %s, PlatformRegion: %s, ClusterRegion: %s, Provider: %s\n", planId, platformRegion, clusterRegion, provider))
						require.NoError(t, err)
						require.NotZero(t, n)

						n, err = newImplementationLog.WriteString(fmt.Sprintf("INPUT: PlanID: %s, PlatformRegion: %s, ClusterRegion: %s, Provider: %s\n", planId, platformRegion, clusterRegion, provider))
						require.NoError(t, err)
						require.NotZero(t, n)
					}

					op := fixOperationWithPlanPlatformRegionAndClusterRegion(planId, platformRegion, clusterRegion, provider)
					err := memoryStorage.Operations().InsertOperation(op)
					assert.NoError(t, err)
					step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProvider, &rules.RulesService{})

					// first run
					increase = true
					_, when, err := step.Run(op, fixLogger())

					require.NoError(t, err)
					require.Zero(t, when)

					for label, count := range queriesCountMap {
						require.NotZero(t, count, "first invocation produced wrong number of queries for label: %s, count: %d", label, count)
					}

					// second run
					increase = false
					// TODO: replace with new hap selection mechanism based on rules configuration
					_, when, err = step.Run(op, fixLogger())

					require.NoError(t, err)
					require.Zero(t, when)

					for label, count := range queriesCountMap {
						require.Zero(t, count, "second invocation produced wrong number of queries for label: %s, count: %d", label, count)
					}

					queriesCountMap = map[string]int{}
				}
			}
		}
	}

	if writeFiles {
		oldImplementationLog.Flush()
		newImplementationLog.Flush()
	}
}

func fixDummySecretBinding(labelsList string) *unstructured.Unstructured {

	unstructuredLabels := make(map[string]interface{})
	labels := strings.Split(labelsList, ",")

	for _, label := range labels {
		pair := strings.Split(label, "=")
		key := pair[0]
		value := pair[1]
		unstructuredLabels[key] = value
	}

	name := uuid.New().String()

	o := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    unstructuredLabels,
			},
			"secretRef": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	o.SetGroupVersionKind(gvk)
	return o
}

func fixOperationWithPlanPlatformRegionAndClusterRegion(planId, platformRegion, clusterRegion string, provider pkg.CloudProvider) internal.Operation {
	o := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	o.ID = uuid.NewString()
	o.ProvisioningParameters.PlatformRegion = platformRegion
	o.ProvisioningParameters.Parameters.Region = &clusterRegion
	o.ProvisioningParameters.PlanID = planId

	return o
}
