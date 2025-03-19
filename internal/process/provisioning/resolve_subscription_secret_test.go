package provisioning

import (
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	tenantName1 = "tenant-1"
	tenantName2 = "tenant-2"

	secretName1 = "secret-1"
	secretName2 = "secret-2"
	secretName3 = "secret-3"
	secretName4 = "secret-4"
	secretName5 = "secret-5"
)

func TestResolveSubscriptionSecretStep(t *testing.T) {
	// given
	operationsStorage := storage.NewMemoryStorage().Operations()
	gardenerClient := createGardenerClient()
	rulesService := createRulesService(t)

	t.Run("", func(t *testing.T) {
		_ = NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService)
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
	sb1 := createSecretBinding(secretBindingName1, namespace, secretName1, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      tenantName1,
	})
	sb2 := createSecretBinding(secretBindingName2, namespace, secretName2, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      tenantName2,
	})
	sb3 := createSecretBinding(secretBindingName3, namespace, secretName3, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
		gardener.EUAccessLabelKey:        "true",
		gardener.SharedLabelKey:          "true",
	})
	sb4 := createSecretBinding(secretBindingName4, namespace, secretName4, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb5 := createSecretBinding(secretBindingName5, namespace, secretName5, map[string]string{
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
				"labels":    labels,
			},
			"secretRef": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
			},
		},
	}
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
				- trial -> S
	`
	tmpfile, err := rules.CreateTempFile(content)
	require.NoError(t, err)
	defer require.NoError(t, os.Remove(tmpfile))

	enabledPlans := &broker.EnablePlans{"aws", "azure", "gcp", "trial"}
	rs, err := rules.NewRulesServiceFromFile(tmpfile, enabledPlans)
	require.NoError(t, err)

	return rs
}
