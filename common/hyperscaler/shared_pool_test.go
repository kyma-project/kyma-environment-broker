package hyperscaler

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSharedPool_SharedCredentialsSecretBinding(t *testing.T) {

	for _, euAccess := range []bool{false, true} {
		for _, testCase := range []struct {
			description    string
			secretBindings []runtime.Object
			shoots         []runtime.Object
			hyperscaler    Type
			expectedSecret string
		}{
			{
				description: "should get only Secret Bindings with proper hyperscaler",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "gcp", true, euAccess),
					newSecretBinding("sb2", "s2", "azure", true, euAccess),
					newSecretBinding("sb3", "s3", "aws", true, euAccess),
					newSecretBinding("sb4", "s4", "openstack_eu-de-1", true, euAccess),
					newSecretBinding("sb5", "s5", "openstack_eu-de-2", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb1"),
					newShoot("sh2", "sb1"),
					newShoot("sh3", "sb1"),
					newShoot("sh4", "sb2"),
				},
				hyperscaler:    GCP(),
				expectedSecret: "s1",
			},
			{
				description: "should get only Secret Bindings with proper hyperscaler and region",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "gcp", true, euAccess),
					newSecretBinding("sb2", "s2", "azure", true, euAccess),
					newSecretBinding("sb3", "s3", "aws", true, euAccess),
					newSecretBinding("sb4", "s4", "openstack_eu-de-1", true, euAccess),
					newSecretBinding("sb5", "s5", "openstack_eu-de-2", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb1"),
					newShoot("sh2", "sb1"),
					newShoot("sh3", "sb1"),
					newShoot("sh4", "sb2"),
				},
				hyperscaler:    Openstack("eu-de-1"),
				expectedSecret: "s4",
			},
			{
				description: "should get only Secret Bindings with proper hyperscaler and region",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "gcp", true, euAccess),
					newSecretBinding("sb2", "s2", "azure", true, euAccess),
					newSecretBinding("sb3", "s3", "aws", true, euAccess),
					newSecretBinding("sb4", "s4", "openstack_eu-de-1", true, euAccess),
					newSecretBinding("sb5", "s5", "openstack_eu-de-2", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb1"),
					newShoot("sh2", "sb1"),
					newShoot("sh3", "sb1"),
					newShoot("sh4", "sb2"),
				},
				hyperscaler:    Openstack("eu-de-2"),
				expectedSecret: "s5",
			},
			{
				description: "should ignore not shared Secret Bindings",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "gcp", true, euAccess),
					newSecretBinding("sb2", "s2", "gcp", false, euAccess),
					newSecretBinding("sb3", "s3", "gcp", false, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb1"),
					newShoot("sh2", "sb1"),
					newShoot("sh3", "sb1"),
					newShoot("sh4", "sb2"),
				},
				hyperscaler:    GCP(),
				expectedSecret: "s1",
			},
			{
				description: "should get least used Secret Binding for GCP",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "gcp", true, euAccess),
					newSecretBinding("sb2", "s2", "gcp", true, euAccess),
					newSecretBinding("sb3", "s3", "gcp", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb1"),
					newShoot("sh2", "sb1"),
					newShoot("sh3", "sb1"),
					newShoot("sh4", "sb2"),
					newShoot("sh5", "sb2"),
					newShoot("sh6", "sb3"),
				},
				hyperscaler:    GCP(),
				expectedSecret: "s3",
			},
			{
				description: "should get least used Secret Binding for Azure",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "azure", true, euAccess),
					newSecretBinding("sb2", "s2", "azure", true, euAccess),
					newSecretBinding("sb3", "s3", "aws", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb1"),
					newShoot("sh2", "sb1"),
					newShoot("sh3", "sb2"),
				},
				hyperscaler:    Azure(),
				expectedSecret: "s2",
			},
			{
				description: "should get least used Secret Binding for AWS",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "aws", true, euAccess),
					newSecretBinding("sb2", "s2", "aws", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb2"),
				},
				hyperscaler:    AWS(),
				expectedSecret: "s1",
			},
			{
				description: "should get the least used Secret Binding for openstack and region eu-de-1",
				secretBindings: []runtime.Object{
					newSecretBinding("sb1", "s1", "gcp", true, euAccess),
					newSecretBinding("sb2", "s2", "azure", true, euAccess),
					newSecretBinding("sb3", "s3", "aws", true, euAccess),
					newSecretBinding("sb4", "s4", "openstack_eu-de-1", true, euAccess),
					newSecretBinding("sb5", "s5", "openstack_eu-de-2", true, euAccess),
					newSecretBinding("sb6", "s6", "openstack_eu-de-1", true, euAccess),
				},
				shoots: []runtime.Object{
					newShoot("sh1", "sb4"),
					newShoot("sh2", "sb4"),
					newShoot("sh3", "sb4"),
					newShoot("sh4", "sb6"),
					newShoot("sh5", "sb6"),
					newShoot("sh6", "sb5"),
				},
				hyperscaler:    Openstack("eu-de-1"),
				expectedSecret: "s6",
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				// given
				gardenerFake := gardener.NewDynamicFakeClient(append(testCase.shoots, testCase.secretBindings...)...)
				pool := NewSharedGardenerAccountPool(gardenerFake, testNamespace)

				// when
				secretBinding, err := pool.SharedCredentialsSecretBinding(testCase.hyperscaler, euAccess)
				require.NoError(t, err)

				// then
				assert.Equal(t, testCase.expectedSecret, secretBinding.GetSecretRefName())
			})
		}
	}
}

func TestSharedPool_SharedCredentialsSecretBinding_Errors(t *testing.T) {
	t.Run("should return error when no Secret Bindings for hyperscaler found", func(t *testing.T) {
		// given
		gardenerFake := gardener.NewDynamicFakeClient(
			newSecretBinding("sb1", "s1", "azure", true, false),
			newSecretBinding("sb2", "s2", "gcp", false, false),
		)
		pool := NewSharedGardenerAccountPool(gardenerFake, testNamespace)

		// when
		_, err := pool.SharedCredentialsSecretBinding(GCP(), false)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no shared secret binding found")
	})
}

func newSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: name, Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"credentials": []byte(name),
		},
	}
}

func newSecretBinding(name, secretName, hyperscaler string, shared bool, euAccess bool) *unstructured.Unstructured {
	secretBinding := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					"hyperscalerType": hyperscaler,
				},
			},
			"secretRef": map[string]interface{}{
				"name":      secretName,
				"namespace": testNamespace,
			},
		},
	}
	secretBinding.SetGroupVersionKind(secretBindingGVK)

	if shared {
		labels := secretBinding.GetLabels()
		labels["shared"] = "true"
		secretBinding.SetLabels(labels)
	}
	applyEuAccess(secretBinding, euAccess)

	return secretBinding
}

func newShoot(name, secretBinding string) *unstructured.Unstructured {
	shoot := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": secretBinding,
			},
		},
	}
	shoot.SetGroupVersionKind(shootGVK)
	return shoot
}
