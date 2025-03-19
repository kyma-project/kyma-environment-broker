package hyperscaler

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

var (
	scheme           = runtime.NewScheme()
	secretBindingGVK = gardener.SecretBindingGVK
	shootGVK         = gardener.ShootGVK
)

const (
	testNamespace = "garden-namespace"
)

func TestCredentialsSecretBinding(t *testing.T) {

	pool := newTestAccountPool()

	var testcases = []struct {
		testDescription           string
		tenantName                string
		hyperscalerType           Type
		expectedSecretBindingName string
		expectedError             string
	}{
		{"In-use credential for tenant1, GCP returns existing secret",
			"tenant1", GCP("cf-jp30"), "secretBinding1", ""},

		{"In-use credential for tenant1, Azure returns existing secret",
			"tenant1", Azure(), "secretBinding2", ""},

		{"In-use credential for tenant2, GCP returns existing secret",
			"tenant2", GCP("cf-jp30"), "secretBinding3", ""},

		{"Available credential for tenant3, AWS labels and returns existing secret",
			"tenant3", GCP("cf-jp30"), "secretBinding4", ""},

		{"Available credential for tenant4, GCP labels and returns existing secret",
			"tenant4", AWS(), "secretBinding5", ""},

		{"There is only dirty Secret for tenant9, Azure labels and returns a new existing secret",
			"tenant9", Azure(), "secretBinding9", ""},

		{"No Available credential for tenant5, Azure returns error",
			"tenant5", Azure(), "",
			"failed to find unassigned secret binding for hyperscalerType: azure"},

		{"No Available credential for tenant6, GCP returns error - ignore secret binding with label shared=true",
			"tenant6", GCP("cf-jp30"), "",
			"failed to find unassigned secret binding for hyperscalerType: gcp"},

		{"Available credential for tenant7, AWS labels and returns existing secret from different namespace",
			"tenant7", AWS(), "secretBinding7", ""},

		{"No Available credential for tenant8, AWS returns error - failed to get referenced secret",
			"tenant8", AWS(), "",
			"failed to find unassigned secret binding for hyperscalerType: aws"},

		{"In-use credential for tenant10, GCP returns existing secret",
			"tenant10", GCP("cf-sa30"), "secretBinding10", ""},

		{"Available credential for tenant11, GCP labels and returns existing secret",
			"tenant11", GCP("cf-sa30"), "secretBinding11", ""},

		{"No Available credential for tenant12, GCP returns error - ignore secret binding with label shared=true",
			"tenant12", GCP("cf-sa30"), "",
			"failed to find unassigned secret binding for hyperscalerType: gcp_cf-sa30"},
	}
	for _, testcase := range testcases {

		t.Run(testcase.testDescription, func(t *testing.T) {
			secretBinding, err := pool.CredentialsSecretBinding(testcase.hyperscalerType, testcase.tenantName, false)
			actualError := ""
			if err != nil {
				actualError = err.Error()
				assert.Equal(t, testcase.expectedError, actualError)
			} else {
				assert.Equal(t, testcase.expectedSecretBindingName, secretBinding.GetName())
				assert.Equal(t, testcase.hyperscalerType.GetKey(), secretBinding.GetLabels()["hyperscalerType"])
				assert.Equal(t, testcase.expectedError, actualError)
			}
		})
	}
}

func TestSecretsAccountPool_IsSecretBindingInternal(t *testing.T) {
	for _, euAccess := range []bool{false, true} {
		t.Run(fmt.Sprintf("EuAccess=%v", euAccess), func(t *testing.T) {
			t.Run("should return true if internal secret binding found", func(t *testing.T) {
				// given
				accPool, _ := newTestAccountPoolWithSecretBindingInternal(euAccess)

				// when
				internal, err := accPool.IsSecretBindingInternal(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.True(t, internal)
			})

			t.Run("should return false if internal secret binding not found", func(t *testing.T) {
				// given
				accPool := newTestAccountPool()

				// when
				internal, err := accPool.IsSecretBindingInternal(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.False(t, internal)
			})

			t.Run("should return false when there is no secret binding in the pool", func(t *testing.T) {
				// given
				accPool := newEmptyTestAccountPool()

				// when
				internal, err := accPool.IsSecretBindingInternal(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.False(t, internal)
			})
		})
	}
}

func TestSecretsAccountPool_IsSecretBindingDirty(t *testing.T) {
	for _, euAccess := range []bool{false, true} {
		t.Run(fmt.Sprintf("EuAccess=%v", euAccess), func(t *testing.T) {
			t.Run("should return true if dirty secret binding found", func(t *testing.T) {
				// given
				accPool, _ := newTestAccountPoolWithSecretBindingDirty(euAccess)

				// when
				isdirty, err := accPool.IsSecretBindingDirty(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.True(t, isdirty)
			})

			t.Run("should return false if dirty secret binding not found", func(t *testing.T) {
				// given
				accPool := newTestAccountPool()

				// when
				isdirty, err := accPool.IsSecretBindingDirty(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.False(t, isdirty)
			})
		})
	}
}

func TestSecretsAccountPool_IsSecretBindingUsed(t *testing.T) {
	for _, euAccess := range []bool{false, true} {
		t.Run(fmt.Sprintf("EuAccess=%v", euAccess), func(t *testing.T) {
			t.Run("should return true when secret binding is in use", func(t *testing.T) {
				// given
				accPool, _ := newTestAccountPoolWithSingleShoot(euAccess)

				// when
				used, err := accPool.IsSecretBindingUsed(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.True(t, used)
			})

			t.Run("should return false when secret binding is not in use", func(t *testing.T) {
				// given
				accPool, _ := newTestAccountPoolWithoutShoots(euAccess)

				// when
				used, err := accPool.IsSecretBindingUsed(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				assert.False(t, used)
			})
		})
	}
}

func TestSecretsAccountPool_MarkSecretBindingAsDirty(t *testing.T) {
	for _, euAccess := range []bool{false, true} {
		t.Run(fmt.Sprintf("EuAccess=%v", euAccess), func(t *testing.T) {
			t.Run("should mark secret binding as dirty", func(t *testing.T) {
				// given
				accPool, gardenerClient := newTestAccountPoolWithoutShoots(euAccess)

				// when
				err := accPool.MarkSecretBindingAsDirty(Azure(), "tenant1", euAccess)

				// then
				require.NoError(t, err)
				secretBinding, err := gardenerClient.Get(context.Background(), "secretBinding1", machineryv1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, secretBinding.GetLabels()["dirty"], "true")
			})
		})
	}
}

func newTestAccountPool() AccountPool {
	secretBinding1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding1",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "gcp",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret1",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding1.SetGroupVersionKind(secretBindingGVK)
	secretBinding2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding2",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret2",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding2.SetGroupVersionKind(secretBindingGVK)
	secretBinding3 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding3",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant2",
					"hyperscalerType": "gcp",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret3",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding3.SetGroupVersionKind(secretBindingGVK)
	secretBinding4 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding4",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": "gcp",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret4",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding4.SetGroupVersionKind(secretBindingGVK)
	secretBinding5 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding5",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": "aws",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret5",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding5.SetGroupVersionKind(secretBindingGVK)
	secretBinding6 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding6",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": "gcp",
					"shared":          "true",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret6",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding6.SetGroupVersionKind(secretBindingGVK)
	secretBinding7 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding7",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": "aws",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret7",
				"namespace": "anothernamespace",
			},
		},
	}
	secretBinding7.SetGroupVersionKind(secretBindingGVK)
	secretBinding8 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding8",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant9",
					"hyperscalerType": "azure",
					"dirty":           "true",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret8",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding8.SetGroupVersionKind(secretBindingGVK)
	secretBinding9 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding9",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": "azure",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret9",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding9.SetGroupVersionKind(secretBindingGVK)
	secretBinding10 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding10",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant10",
					"hyperscalerType": "gcp_cf-sa30",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret10",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding10.SetGroupVersionKind(secretBindingGVK)
	secretBinding11 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding11",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": "gcp_cf-sa30",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret11",
				"namespace": testNamespace,
			},
		},
	}
	secretBinding11.SetGroupVersionKind(secretBindingGVK)
	gardenerFake := gardener.NewDynamicFakeClient(secretBinding1, secretBinding2, secretBinding3, secretBinding4,
		secretBinding5, secretBinding6, secretBinding7, secretBinding8, secretBinding9, secretBinding10, secretBinding11)

	return NewAccountPool(gardenerFake, testNamespace)
}

func newTestAccountPoolWithSingleShoot(euAccess bool) (AccountPool, dynamic.ResourceInterface) {
	secretBinding1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding1",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret1",
				"namespace": testNamespace,
			},
		},
	}
	applyEuAccess(secretBinding1, euAccess)
	secretBinding1.SetGroupVersionKind(secretBindingGVK)

	shoot1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "shoot1",
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": "secretBinding1",
			},
			"status": map[string]interface{}{
				"lastOperation": map[string]interface{}{
					"state": "Succeeded",
					"type":  "Reconcile",
				},
			},
		},
	}
	shoot1.SetGroupVersionKind(shootGVK)

	gardenerFake := gardener.NewDynamicFakeClient(shoot1, secretBinding1)
	return NewAccountPool(gardenerFake, testNamespace), gardenerFake.Resource(gardener.SecretBindingResource).Namespace(testNamespace)
}

func newEmptyTestAccountPool() AccountPool {
	secretBinding1 := &unstructured.Unstructured{}
	secretBinding1.SetGroupVersionKind(secretBindingGVK)
	gardenerFake := gardener.NewDynamicFakeClient(secretBinding1)
	return NewAccountPool(gardenerFake, testNamespace)
}

func applyEuAccess(obj *unstructured.Unstructured, euAccess bool) {
	if euAccess {
		labels := obj.GetLabels()
		labels["euAccess"] = "true"
		obj.SetLabels(labels)
	}
}

func newTestAccountPoolWithSecretBindingInternal(euAccess bool) (AccountPool, dynamic.ResourceInterface) {
	secretBinding1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding1",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
					"internal":        "true",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret1",
				"namespace": testNamespace,
			},
		},
	}
	applyEuAccess(secretBinding1, euAccess)
	secretBinding1.SetGroupVersionKind(secretBindingGVK)

	gardenerFake := gardener.NewDynamicFakeClient(secretBinding1)
	return NewAccountPool(gardenerFake, testNamespace), gardenerFake.Resource(gardener.SecretBindingResource).Namespace(testNamespace)
}

func newTestAccountPoolWithSecretBindingDirty(euAccess bool) (AccountPool, dynamic.ResourceInterface) {
	secretBinding1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding1",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
					"dirty":           "true",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret1",
				"namespace": testNamespace,
			},
		},
	}
	applyEuAccess(secretBinding1, euAccess)
	secretBinding1.SetGroupVersionKind(secretBindingGVK)

	shoot1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "shoot1",
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": "secretBinding1",
			},
			"status": map[string]interface{}{
				"lastOperation": map[string]interface{}{
					"state": "Succeeded",
					"type":  "Reconcile",
				},
			},
		},
	}
	shoot1.SetGroupVersionKind(shootGVK)

	gardenerFake := gardener.NewDynamicFakeClient(shoot1, secretBinding1)
	return NewAccountPool(gardenerFake, testNamespace), gardenerFake.Resource(gardener.SecretBindingResource).Namespace(testNamespace)
}

func newTestAccountPoolWithShootsUsingSecretBinding(euAccess bool) (AccountPool, dynamic.ResourceInterface) {
	secretBinding1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding1",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret1",
				"namespace": testNamespace,
			},
		},
	}
	applyEuAccess(secretBinding1, euAccess)
	secretBinding1.SetGroupVersionKind(secretBindingGVK)

	shoot1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "shoot1",
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": "secretBinding1",
			},
			"status": map[string]interface{}{
				"lastOperation": map[string]interface{}{
					"state": "Succeeded",
					"type":  "Reconcile",
				},
			},
		},
	}
	shoot1.SetGroupVersionKind(shootGVK)

	shoot2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "shoot2",
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": "secretBinding1",
			},
			"status": map[string]interface{}{
				"lastOperation": map[string]interface{}{
					"state": "Succeeded",
					"type":  "Reconcile",
				},
			},
		},
	}
	shoot2.SetGroupVersionKind(shootGVK)

	gardenerFake := gardener.NewDynamicFakeClient(shoot1, shoot2, secretBinding1)
	return NewAccountPool(gardenerFake, testNamespace), gardenerFake.Resource(gardener.SecretBindingResource).Namespace(testNamespace)
}

func newTestAccountPoolWithoutShoots(euAccess bool) (AccountPool, dynamic.ResourceInterface) {
	secretBinding1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "secretBinding1",
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"tenantName":      "tenant1",
					"hyperscalerType": "azure",
				},
			},
			"secretRef": map[string]interface{}{
				"name":      "secret1",
				"namespace": testNamespace,
			},
		},
	}
	applyEuAccess(secretBinding1, euAccess)
	secretBinding1.SetGroupVersionKind(secretBindingGVK)

	gardenerFake := gardener.NewDynamicFakeClient(secretBinding1)
	return NewAccountPool(gardenerFake, testNamespace), gardenerFake.Resource(gardener.SecretBindingResource).Namespace(testNamespace)
}
