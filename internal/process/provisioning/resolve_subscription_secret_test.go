package provisioning

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	awsTenantName   = "aws-tenant-1"
	azureTenantName = "azure-tenant-2"

	awsEUAccessClaimedSecretName   = "aws-euaccess-tenant-1"
	azureEUAccessClaimedSecretName = "azure-euaccess-tenant-2"
	gcpEUAccessSharedSecretName    = "gcp-euaccess-shared"
	awsMostUsedSharedSecretName    = "aws-most-used-shared"
	awsLeastUsedSharedSecretName   = "aws-least-used-shared"
)

func TestResolveSubscriptionSecretStep(t *testing.T) {
	// given
	operationsStorage := storage.NewMemoryStorage().Operations()
	gardenerClient := createGardenerClient()
	rulesService := createRulesService(t)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	t.Run("should resolve secret name for aws hyperscaler and existing tenant", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-1"
			instanceID     = "instance-1"
			platformRegion = "cf-eu11"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.AWS)
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = awsTenantName
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, awsEUAccessClaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)
	})
}

func createGardenerClient() *gardener.Client {
	const (
		namespace          = "test"
		secretBindingName1 = "secret-binding-1"
		secretBindingName2 = "secret-binding-2"
		secretBindingName3 = "secret-binding-3"
		secretBindingName4 = "secret-binding-4"
		secretBindingName5 = "secret-binding-5"
	)
	sb1 := createSecretBinding(secretBindingName1, namespace, awsEUAccessClaimedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      awsTenantName,
	})
	sb2 := createSecretBinding(secretBindingName2, namespace, azureEUAccessClaimedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      azureTenantName,
	})
	sb3 := createSecretBinding(secretBindingName3, namespace, gcpEUAccessSharedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
		gardener.EUAccessLabelKey:        "true",
		gardener.SharedLabelKey:          "true",
	})
	sb4 := createSecretBinding(secretBindingName4, namespace, awsMostUsedSharedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb5 := createSecretBinding(secretBindingName5, namespace, awsLeastUsedSharedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	shoot1 := createShoot("shoot-1", namespace, secretBindingName4)
	shoot2 := createShoot("shoot-2", namespace, secretBindingName4)
	shoot3 := createShoot("shoot-3", namespace, secretBindingName5)

	fakeGardenerClient := gardener.NewDynamicFakeClient(sb1, sb2, sb3, sb4, sb5, shoot1, shoot2, shoot3)

	return gardener.NewClient(fakeGardenerClient, namespace)
}

func createSecretBinding(name, namespace, secretName string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"secretRef": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
			},
		},
	}
	u.SetLabels(labels)
	u.SetGroupVersionKind(gardener.SecretBindingGVK)

	return u
}

func createShoot(name, namespace, secretBindingName string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": secretBindingName,
			},
			"status": map[string]interface{}{
				"lastOperation": map[string]interface{}{
					"state": "Succeeded",
					"type":  "Reconcile",
				},
			},
		},
	}
	u.SetGroupVersionKind(gardener.ShootGVK)

	return u
}

func createRulesService(t *testing.T) *rules.RulesService {
	content := `rule:
                      - aws(PR=cf-eu11) -> EU
                      - azure(PR=cf-ch20) -> EU
                      - gcp(PR=cf-eu30) -> EU,S
                      - trial -> S`
	tmpfile, err := rules.CreateTempFile(content)
	require.NoError(t, err)
	defer os.Remove(tmpfile)

	enabledPlans := &broker.EnablePlans{"aws", "azure", "gcp", "trial"}
	rs, err := rules.NewRulesServiceFromFile(tmpfile, enabledPlans)
	require.NoError(t, err)

	return rs
}
