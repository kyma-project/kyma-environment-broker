package provisioning

import (
	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	k8sTesting "k8s.io/client-go/testing"
	"strings"
	"testing"
)

func TestSecretBindingLabelSelector(t *testing.T) {
	/**
	This test checks if the labe selector used by new stap implementation is the same as the old one producees.
	*/
	for _, platformRegion := range []string{"cf-eu10", "cf-eu11", "cf-ch20", "cf-sa30"} {
		t.Run(platformRegion, func(t *testing.T) {
			for _, tc := range []struct {
				providerType string
				planID       string
			}{
				{providerType: "aws", planID: broker.AWSPlanID},
				{providerType: "azure", planID: broker.AzurePlanID},
				{providerType: "gcp", planID: broker.GCPPlanID},
				{providerType: "aws", planID: broker.TrialPlanID},
				{providerType: "azure", planID: broker.TrialPlanID},
				{providerType: "openstack", planID: broker.SapConvergedCloudPlanID},
				{providerType: "aws", planID: broker.FreemiumPlanID},
			} {
				planName := broker.PlanNamesMapping[tc.planID]
				t.Run(planName, func(t *testing.T) {
					var referenceSelector *string = ptr.String("")
					var gotSelector *string = ptr.String("")

					// given
					operation := fixProvisioningOperation("westeurope", platformRegion, tc.planID, tc.providerType)
					step := fixResolveCredentialStep(t, referenceSelector, operation)

					_, backoff, _ := step.Run(operation, fixLogger())
					// after the old step is run - we have referenceSelector with a reference value, which is used for an assertion

					require.Zero(t, backoff)
					newStep := fixNewResolveStep(t, gotSelector, operation)

					// when
					_, backoff, _ = newStep.Run(operation, fixLogger())
					require.Zero(t, backoff)

					// then
					assertSelectors(t, referenceSelector, gotSelector)
				})
			}
		})
	}
}

func assertSelectors(t *testing.T, expected *string, got *string) {
	t.Helper()
	t.Log("expectedSet: ", *expected)
	expectedParts := strings.Split(*expected, ",")
	expectedSet := sets.New(expectedParts...)

	gotParts := strings.Split(*got, ",")
	gotSet := sets.New(gotParts...)

	assert.Equal(t, expectedSet, gotSet)
}

func dummySecretBinding() gardener.SecretBinding {
	name := uuid.New().String()
	sb := gardener.SecretBinding{}
	sb.SetName(name)
	sb.SetNamespace(namespace)
	sb.SetSecretRefName(name)
	return sb
}

func savingLabelSelectorReactor(selector *string) func(action k8sTesting.Action) (bool, runtime.Object, error) {
	return func(action k8sTesting.Action) (bool, runtime.Object, error) {
		labelSelector := action.(k8sTesting.ListActionImpl).GetListRestrictions().Labels.String()
		*selector = labelSelector

		labels := map[string]string{}
		requirements, _ := action.(k8sTesting.ListActionImpl).GetListRestrictions().Labels.Requirements()
		for _, r := range requirements {
			if len(r.Values()) > 0 {
				labels[r.Key()] = r.Values().List()[0]
			}
		}
		sb := dummySecretBinding()
		sb.SetLabels(labels)
		listToReturn := &unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{sb.Unstructured},
		}
		return true, listToReturn, nil
	}
}

func fixProvisioningOperation(region string, platformRegion string, planID, provider string) internal.Operation {
	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ProvisioningParameters.Parameters.Region = ptr.String(region)
	operation.ProvisioningParameters.PlatformRegion = platformRegion
	operation.ProvisioningParameters.PlanID = planID
	operation.ProviderValues = &internal.ProviderValues{
		ProviderType: provider,
	}
	return operation
}

func fixResolveCredentialStep(t *testing.T, selector *string, operation internal.Operation) *ResolveCredentialsStep {
	gardenerK8sClient := gardener.NewDynamicFakeClient()
	gardenerK8sClient.PrependReactor("list",
		gardener.SecretBindingResource.Resource,
		savingLabelSelectorReactor(selector))
	memoryStorage := storage.NewMemoryStorage()
	accountProvider := hyperscaler.NewAccountProvider(hyperscaler.NewAccountPool(gardenerK8sClient, namespace), hyperscaler.NewSharedGardenerAccountPool(gardenerK8sClient, namespace))
	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProvider, &rules.RulesService{})
	err := memoryStorage.Operations().InsertOperation(operation)
	require.NoError(t, err)
	return step
}

func fixNewResolveStep(t *testing.T, selector *string, operation internal.Operation) process.Step {
	// todo: implement creation of a new step using similar gardener client like in fixResolveCredentialStep (see - prepend reactor)
	return fixResolveCredentialStep(t, selector, operation)
}
