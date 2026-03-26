package broker

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type provisionEndpointBuilder struct {
	brokerConfig                         Config
	gardenerConfig                       gardener.Config
	imConfig                             InfrastructureManager
	db                                   storage.BrokerStorage
	queue                                Queue
	plansConfig                          PlansConfig
	log                                  *slog.Logger
	dashboardConfig                      dashboard.Config
	kcBuilder                            kubeconfig.KcBuilder
	freemiumWhitelist                    whitelist.Set
	schemaService                        *SchemaService
	providerSpec                         ConfigurationProvider
	valuesProvider                       ValuesProvider
	providerConfigProvider               config.ConfigMapConfigProvider
	quotaClient                          QuotaClient
	quotaWhitelist                       whitelist.Set
	rulesService                         *rules.RulesService
	gardenerClient                       *gardener.Client
	awsClientFactory                     aws.ClientFactory
	btpRegionsMigrationSapConvergedCloud map[string]string
}

func newProvisionEndpointBuilder() *provisionEndpointBuilder {
	return &provisionEndpointBuilder{}
}

func (b *provisionEndpointBuilder) WithInfrastructureManager(im InfrastructureManager) *provisionEndpointBuilder {
	b.imConfig = im
	return b
}

func (b *provisionEndpointBuilder) WithStorage(st storage.BrokerStorage) *provisionEndpointBuilder {
	b.db = st
	return b
}

func (b *provisionEndpointBuilder) WithLogger(l *slog.Logger) *provisionEndpointBuilder {
	b.log = l
	return b
}

func (b *provisionEndpointBuilder) WithSchemaService(s *SchemaService) *provisionEndpointBuilder {
	b.schemaService = s
	return b
}

func (b *provisionEndpointBuilder) Build() *ProvisionEndpoint {
	return NewProvision(
		b.brokerConfig,
		b.gardenerConfig,
		b.imConfig,
		b.db,
		b.queue,
		b.plansConfig,
		b.log,
		b.dashboardConfig,
		b.kcBuilder,
		b.freemiumWhitelist,
		b.schemaService,
		b.providerSpec,
		b.valuesProvider,
		b.providerConfigProvider,
		b.quotaClient,
		b.quotaWhitelist,
		b.rulesService,
		b.gardenerClient,
		b.awsClientFactory,
		b.btpRegionsMigrationSapConvergedCloud,
	)
}

func TestColocateControlPlane(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	st := storage.NewMemoryStorage()
	imConfig := InfrastructureManager{
		IngressFilteringPlans: []string{"aws", "azure", "gcp"},
	}
	provisionEndpoint := newProvisionEndpointBuilder().
		WithStorage(st).
		WithInfrastructureManager(imConfig).
		WithLogger(log).
		WithSchemaService(createSchemaService(t)).
		Build()

	t.Run("should parse colocateControlPlane: true", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "colocateControlPlane": true }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

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
	provisionEndpoint := newProvisionEndpointBuilder().
		WithStorage(st).
		WithInfrastructureManager(imConfig).
		WithLogger(log).
		WithSchemaService(createSchemaService(t)).
		Build()

	t.Run("should parse gvisor enabled: true", func(t *testing.T) {
		// given
		rawParameters := json.RawMessage(`{ "gvisor": { "enabled": true } }`)
		details := domain.ProvisionDetails{
			RawParameters: rawParameters,
		}

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

		// when
		parameters, err := provisionEndpoint.extractInputParameters(details)

		// then
		require.NoError(t, err)
		require.Len(t, parameters.AdditionalWorkerNodePools, 1)
		require.NotNil(t, parameters.AdditionalWorkerNodePools[0].Gvisor)
		assert.True(t, parameters.AdditionalWorkerNodePools[0].Gvisor.Enabled)
	})
}
