package broker

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColocateControlPlane(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	st := storage.NewMemoryStorage()
	imConfig := InfrastructureManager{
		IngressFilteringPlans: []string{"aws", "azure", "gcp"},
	}

	t.Run("should parse colocateControlPlane: true", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "colocateControlPlane": true }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.True(t, *parameters.ColocateControlPlane)
	})

	t.Run("should parse colocateControlPlane: false", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "colocateControlPlane": false }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.False(t, *parameters.ColocateControlPlane)
	})

	t.Run("shouldn't parse nil colocateControlPlane", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}
		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.Nil(t, parameters.ColocateControlPlane)
	})

}

func TestGvisorProvisioningParameters(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	st := storage.NewMemoryStorage()
	imConfig := InfrastructureManager{
		IngressFilteringPlans: []string{"aws", "azure", "gcp"},
	}

	t.Run("should parse gvisor enabled: true", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "gvisor": { "enabled": true } }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		require.NotNil(t, parameters.Gvisor)
		assert.True(t, parameters.Gvisor.Enabled)
	})

	t.Run("should not parse gvisor when key is absent", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{}`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.Nil(t, parameters.Gvisor)
	})

	t.Run("should parse gvisor enabled: false", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "gvisor": { "enabled": false } }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		require.NotNil(t, parameters.Gvisor)
		assert.False(t, parameters.Gvisor.Enabled)
	})

	t.Run("should parse gvisor in additionalWorkerNodePools item", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{
			"additionalWorkerNodePools": [
				{
					"name": "worker-1",
					"machineType": "m5.xlarge",
					"haZones": false,
					"autoScalerMin": 1,
					"autoScalerMax": 3,
					"gvisor": { "enabled": true }
				}
			]
		}`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

		provisionEndpoint := NewProvision(
			Config{},
			gardener.Config{},
			imConfig,
			st,
			nil,
			nil,
			log,
			dashboard.Config{},
			nil,
			nil,
			createSchemaService(t),
			nil,
			nil,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		require.Len(t, parameters.AdditionalWorkerNodePools, 1)
		require.NotNil(t, parameters.AdditionalWorkerNodePools[0].Gvisor)
		assert.True(t, parameters.AdditionalWorkerNodePools[0].Gvisor.Enabled)
	})
}
