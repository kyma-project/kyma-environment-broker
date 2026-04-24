package provider

import (
	"context"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNodemeterYAML = `
meters:
  node:
    machine_types:
      aws:
        m6i.4xlarge:
          nodes: 8
          default_volume_size: "84Gi"
        m6i.8xlarge:
          nodes: 16
          default_volume_size: "148Gi"
      azure:
        standard_d4s_v5:
          nodes: 2
          default_volume_size: "80Gi"
        standard_d32s_v5:
          nodes: 16
          default_volume_size: "148Gi"
      gcp:
        n2-standard-4:
          nodes: 2
          default_volume_size: "80Gi"
      alicloud:
        ecs.g9i.large:
          nodes: 1.5
          default_volume_size: "80Gi"
      openstack:
        g_c4_m16:
          nodes: 2
          default_volume_size: "80Gi"
`

func newTestConfigMap(nodemeterYAML string) *coreV1.ConfigMap {
	return &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "consumption-reporter-config",
			Namespace: "kcp-system",
		},
		Data: map[string]string{
			kcrNodemeterKey: nodemeterYAML,
		},
	}
}

func TestKCRVolumeProvider_DefaultVolumeSizeGb(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithObjects(newTestConfigMap(testNodemeterYAML)).Build()
	p := NewKCRVolumeProvider(k8sClient, "consumption-reporter-config")

	tests := []struct {
		name      string
		provider  pkg.CloudProvider
		machine   string
		wantSize  int
		wantError bool
	}{
		{"AWS m6i.4xlarge", pkg.AWS, "m6i.4xlarge", 84, false},
		{"AWS m6i.8xlarge", pkg.AWS, "m6i.8xlarge", 148, false},
		{"Azure Standard_D4s_v5 (case insensitive)", pkg.Azure, "Standard_D4s_v5", 80, false},
		{"Azure Standard_D32s_v5", pkg.Azure, "Standard_D32s_v5", 148, false},
		{"GCP n2-standard-4", pkg.GCP, "n2-standard-4", 80, false},
		{"Alicloud ecs.g9i.large", pkg.Alicloud, "ecs.g9i.large", 80, false},
		{"SapConvergedCloud g_c4_m16", pkg.SapConvergedCloud, "g_c4_m16", 80, false},
		{"unknown machine", pkg.AWS, "m99.unknown", 0, true},
		{"unknown provider", pkg.UnknownProvider, "m6i.4xlarge", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := p.DefaultVolumeSizeGb(context.Background(), tt.provider, tt.machine)
			if tt.wantError {
				require.Error(t, err)
				assert.False(t, kebError.IsTemporaryError(err), "unexpected machine lookup errors should not be temporary")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantSize, size)
			}
		})
	}
}

func TestKCRVolumeProvider_DefaultVolumeSizeGb_ConfigMapMissing(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()
	p := NewKCRVolumeProvider(k8sClient, "consumption-reporter-config")

	_, err := p.DefaultVolumeSizeGb(context.Background(), pkg.AWS, "m6i.4xlarge")
	require.Error(t, err)
	assert.True(t, kebError.IsTemporaryError(err), "missing ConfigMap should return a temporary error to trigger retry")
}

func TestKCRVolumeProvider_DefaultVolumeSizeGb_MissingNodemeterKey(t *testing.T) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "consumption-reporter-config", Namespace: "kcp-system"},
		Data:       map[string]string{"other-key": "value"},
	}
	k8sClient := fake.NewClientBuilder().WithObjects(cm).Build()
	p := NewKCRVolumeProvider(k8sClient, "consumption-reporter-config")

	_, err := p.DefaultVolumeSizeGb(context.Background(), pkg.AWS, "m6i.4xlarge")
	require.Error(t, err)
	assert.False(t, kebError.IsTemporaryError(err))
}

func TestKCRVolumeProvider_ValidateAllMachineTypes(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithObjects(newTestConfigMap(testNodemeterYAML)).Build()
	p := NewKCRVolumeProvider(k8sClient, "consumption-reporter-config")

	t.Run("all valid", func(t *testing.T) {
		machines := map[pkg.CloudProvider][]string{
			pkg.AWS:   {"m6i.4xlarge", "m6i.8xlarge"},
			pkg.Azure: {"Standard_D4s_v5"},
		}
		require.NoError(t, p.ValidateAllMachineTypes(context.Background(), machines))
	})

	t.Run("missing machine type", func(t *testing.T) {
		machines := map[pkg.CloudProvider][]string{
			pkg.AWS: {"m6i.4xlarge", "m99.unknown"},
		}
		require.Error(t, p.ValidateAllMachineTypes(context.Background(), machines))
	})
}

func TestParseNodemeterYAML(t *testing.T) {
	data, err := parseNodemeterYAML(testNodemeterYAML)
	require.NoError(t, err)

	assert.Equal(t, 84, data["aws"]["m6i.4xlarge"])
	assert.Equal(t, 148, data["aws"]["m6i.8xlarge"])
	assert.Equal(t, 80, data["azure"]["standard_d4s_v5"])
	assert.Equal(t, 80, data["gcp"]["n2-standard-4"])
}

func TestParseNodemeterYAML_InvalidSize(t *testing.T) {
	badYAML := `
meters:
  node:
    machine_types:
      aws:
        m6i.4xlarge:
          default_volume_size: "not-a-number-Gi"
`
	_, err := parseNodemeterYAML(badYAML)
	require.Error(t, err)
}
