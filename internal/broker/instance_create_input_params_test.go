package broker

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShootAndSeedSameRegion(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	st := storage.NewMemoryStorage()
	imConfig := InfrastructureManager{
		EnableIngressFiltering: true,
		IngressFilteringPlans:  []string{"aws", "azure", "gcp"},
	}

	t.Run("should parse shootAndSeedSameRegion - true", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "shootAndSeedSameRegion": true }`)
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
			false,
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.True(t, *parameters.ShootAndSeedSameRegion)
	})

	t.Run("should parse shootAndSeedSameRegion - false", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "shootAndSeedSameRegion": false }`)
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
			false,
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.False(t, *parameters.ShootAndSeedSameRegion)
	})

	t.Run("should parse shootAndSeedSameRegion - nil", func(t *testing.T) {
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
			false,
		)

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		assert.Nil(t, parameters.ShootAndSeedSameRegion)
	})

}
