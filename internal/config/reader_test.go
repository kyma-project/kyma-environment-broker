package config_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	configMapName    = "keb-config"
	defaultConfigKey = "default"
)

func TestConfigReaderSuccessFlow(t *testing.T) {
	// setup
	ctx := context.TODO()
	cfgMap, err := fixConfigMap()
	require.NoError(t, err)

	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(cfgMap).Build()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, log)

	t.Run("should read default KEB config", func(t *testing.T) {
		// when
		rawCfg, err := cfgReader.Read(configMapName, broker.AWSPlanName)

		// then
		require.NoError(t, err)
		assert.Equal(t, cfgMap.Data[defaultConfigKey], rawCfg)
	})

	t.Run("should read KEB config for azure plan", func(t *testing.T) {
		// when
		rawCfg, err := cfgReader.Read(configMapName, broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.Equal(t, cfgMap.Data[broker.AzurePlanName], rawCfg)
	})

	t.Run("should read KEB config for Kyma trial plan", func(t *testing.T) {
		// when
		rawCfg, err := cfgReader.Read(configMapName, broker.TrialPlanName)

		// then
		require.NoError(t, err)
		assert.Equal(t, cfgMap.Data[broker.TrialPlanName], rawCfg)
	})
}

func TestConfigReaderErrors(t *testing.T) {
	// setup
	ctx := context.TODO()

	k8sClient := failingK8sClient{}
	fakeK8sClient := fake.NewClientBuilder().Build()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	t.Run("should return error while fetching configmap on List() of K8s client", func(t *testing.T) {
		// given
		cfgReader := config.NewConfigMapReader(ctx, k8sClient, log)

		// when
		rawCfg, err := cfgReader.Read(configMapName, broker.AzurePlanName)

		// then
		require.Error(t, err)
		log.Error(err.Error())
		assert.Equal(t, "", rawCfg)
	})

	t.Run("should return error while getting config string for a plan", func(t *testing.T) {
		// given
		cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, log)

		// when
		rawCfg, err := cfgReader.Read(configMapName, broker.AzurePlanName)

		// then
		require.Error(t, err)
		log.Error(err.Error())
		assert.Equal(t, "", rawCfg)
	})
}

func fixConfigMap() (*coreV1.ConfigMap, error) {
	yamlFilePath := path.Join("testdata", fmt.Sprintf("%s.yaml", configMapName))
	contents, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("while reading configmap")
	}

	contents = bytes.ReplaceAll(contents, []byte("Kind"), []byte("kind"))
	var tempCfgMap tempConfigMap
	err = yaml.Unmarshal(contents, &tempCfgMap)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling configmap")
	}

	return tempCfgMap.toConfigMap(), nil
}

type tempConfigMap struct {
	APIVersion string            `yaml:"apiVersion,omitempty"`
	Kind       string            `yaml:",omitempty"`
	Metadata   tempMetadata      `yaml:"metadata,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
}

type tempMetadata struct {
	Name      string            `yaml:"name,omitempty"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

func (m *tempConfigMap) toConfigMap() *coreV1.ConfigMap {
	return &coreV1.ConfigMap{
		TypeMeta: metaV1.TypeMeta{
			Kind:       m.Kind,
			APIVersion: m.APIVersion,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      m.Metadata.Name,
			Namespace: m.Metadata.Namespace,
			Labels:    m.Metadata.Labels,
		},
		Data: m.Data,
	}
}

type failingK8sClient struct{}

func (c failingK8sClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return fmt.Errorf("not implemented")
}

func (c failingK8sClient) Status() client.StatusWriter {
	panic("not implemented")
}

func (c failingK8sClient) Scheme() *runtime.Scheme {
	panic("not implemented")
}

func (c failingK8sClient) RESTMapper() meta.RESTMapper {
	panic("not implemented")
}

func (c failingK8sClient) SubResource(s string) client.SubResourceClient {
	panic("not implemented")
}

func (c failingK8sClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, fmt.Errorf("not implemented")
}

func (c failingK8sClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return false, fmt.Errorf("not implemented")
}
