package config_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const wrongConfigPlan = "wrong"

func TestConfigProvider(t *testing.T) {
	// setup
	ctx := context.TODO()
	cfgMap, err := fixConfigMap()
	require.NoError(t, err)

	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(cfgMap).Build()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, log)
	cfgValidator := config.NewConfigMapKeysValidator()
	cfgConverter := config.NewConfigMapConverter()
	cfgProvider := config.NewConfigProvider(cfgReader, cfgValidator, cfgConverter)

	t.Run("should provide config for azure plan", func(t *testing.T) {
		// given
		cfg := &internal.ConfigForPlan{}
		expectedCfg := fixAzureConfig()

		// when
		err := cfgProvider.Provide(configMapName, broker.AzurePlanName, config.RuntimeConfigurationRequiredFields, cfg)

		// then
		require.NoError(t, err)
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("should provide config for a default", func(t *testing.T) {
		// given
		cfg := &internal.ConfigForPlan{}
		expectedCfg := fixDefault()

		// when
		err := cfgProvider.Provide(configMapName, broker.AWSPlanName, config.RuntimeConfigurationRequiredFields, cfg)

		// then
		require.NoError(t, err)
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("validator should return error indicating missing required fields", func(t *testing.T) {
		// given
		cfg := &internal.ConfigForPlan{}
		expectedMissingConfigKeys := []string{
			"kyma-template",
		}
		expectedErrMsg := fmt.Sprintf("missing required configuration entires: %s", strings.Join(expectedMissingConfigKeys, ","))

		// when
		err := cfgProvider.Provide(configMapName, wrongConfigPlan, config.RuntimeConfigurationRequiredFields, cfg)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, expectedErrMsg)
		assert.Equal(t, internal.ConfigForPlan{}, *cfg)
	})

	t.Run("reader should return error indicating missing configmap", func(t *testing.T) {
		// given
		cfg := &internal.ConfigForPlan{}
		err = fakeK8sClient.Delete(ctx, cfgMap)
		require.NoError(t, err)

		// when
		err := cfgProvider.Provide(configMapName, broker.AzurePlanName, config.RuntimeConfigurationRequiredFields, cfg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configmap keb-config does not exist in kcp-system namespace")
		assert.Equal(t, internal.ConfigForPlan{}, *cfg)
	})
}

func fixAzureConfig() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{
		KymaTemplate: `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: tbd
  namespace: kyma-system
spec:
  sync:
    strategy: secret
  channel: fast
  modules: []
  additional-components:
  - name: "additional-component1"
    namespace: "kyma-system"
  - name: "additional-component2"
    namespace: "test-system"
  # no additional-component3
  - name: "azure-component"
  namespace: "azure-system"
  source:
    url: "https://azure.domain/component/azure-component.git"`,
	}
}

func fixDefault() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{
		KymaTemplate: `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma1
  namespace: kyma-system
spec:
  channel: fast
  modules:
  - name: istio`,
	}
}
