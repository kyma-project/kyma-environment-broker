package config_test

import (
	"context"
	"errors"
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
	cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, log, "keb-config")
	cfgValidator := config.NewConfigMapKeysValidator()
	cfgConverter := config.NewConfigMapConverter()
	cfgProvider := config.NewConfigProvider(cfgReader, cfgValidator, cfgConverter)

	t.Run("should provide config for Kyma 2.4.0 azure plan", func(t *testing.T) {
		// given
		expectedCfg := fixAzureConfig()
		// when
		cfg, err := cfgProvider.ProvideForGivenPlan(broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("should provide config for a default", func(t *testing.T) {
		// given
		expectedCfg := fixDefault()
		// when
		cfg, err := cfgProvider.ProvideForGivenPlan(broker.AWSPlanName)

		// then
		require.NoError(t, err)
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("validator should return error indicating missing required fields", func(t *testing.T) {
		// given
		expectedMissingConfigKeys := []string{
			"kyma-template",
		}
		expectedErrMsg := fmt.Sprintf("missing required configuration entires: %s", strings.Join(expectedMissingConfigKeys, ","))
		// when
		cfg, err := cfgProvider.ProvideForGivenPlan(wrongConfigPlan)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, expectedErrMsg)
		assert.Nil(t, cfg)
	})

	t.Run("reader should return error indicating missing configmap", func(t *testing.T) {
		// given
		err = fakeK8sClient.Delete(ctx, cfgMap)
		require.NoError(t, err)

		// when
		cfg, err := cfgProvider.ProvideForGivenPlan(broker.AzurePlanName)

		// then
		require.Error(t, err)
		assert.Equal(t, "configmap keb-config with configuration does not exist", errors.Unwrap(err).Error())
		assert.Nil(t, cfg)
	})
}

func fixAzureConfig() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{}
}

func fixDefault() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{
		KymaTemplate: `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma1
  namespace: kyma-system
spec:
  channel: stable
  modules:
  - name: istio`,
	}
}
