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

type fakeProvisionEndpointBuilder struct {
	brokerConfig           Config
	gardenerConfig         gardener.Config
	imConfig               InfrastructureManager
	db                     storage.BrokerStorage
	queue                  Queue
	plansConfig            PlansConfig
	log                    *slog.Logger
	dashboardConfig        dashboard.Config
	kcBuilder              kubeconfig.KcBuilder
	freemiumWhitelist      whitelist.Set
	gvisorWhitelist        whitelist.Set
	schemaService          *SchemaService
	providerSpec           ConfigurationProvider
	valuesProvider         ValuesProvider
	providerConfigProvider config.ConfigMapConfigProvider
	quotaClient            QuotaClient
	quotaWhitelist         whitelist.Set
	rulesService           *rules.RulesService
	gardenerClient         *gardener.Client
	awsClientFactory       aws.ClientFactory
}

func NewFakeProvisionEndpointBuilder() *fakeProvisionEndpointBuilder {
	return &fakeProvisionEndpointBuilder{}
}

func (b *fakeProvisionEndpointBuilder) WithInfrastructureManager(im InfrastructureManager) *fakeProvisionEndpointBuilder {
	b.imConfig = im
	return b
}

func (b *fakeProvisionEndpointBuilder) WithStorage(st storage.BrokerStorage) *fakeProvisionEndpointBuilder {
	b.db = st
	return b
}

func (b *fakeProvisionEndpointBuilder) WithLogger(l *slog.Logger) *fakeProvisionEndpointBuilder {
	b.log = l
	return b
}

func (b *fakeProvisionEndpointBuilder) WithSchemaService(s *SchemaService) *fakeProvisionEndpointBuilder {
	b.schemaService = s
	return b
}

func (b *fakeProvisionEndpointBuilder) WithConfig(brokerConfig Config) *fakeProvisionEndpointBuilder {
	b.brokerConfig = brokerConfig
	return b
}

func (b *fakeProvisionEndpointBuilder) WithGardenerConfig(gardenerConfig gardener.Config) *fakeProvisionEndpointBuilder {
	b.gardenerConfig = gardenerConfig
	return b
}

func (b *fakeProvisionEndpointBuilder) WithQueue(queue Queue) *fakeProvisionEndpointBuilder {
	b.queue = queue
	return b
}

func (b *fakeProvisionEndpointBuilder) WithDashboardConfig(dashboardConfig dashboard.Config) *fakeProvisionEndpointBuilder {
	b.dashboardConfig = dashboardConfig
	return b
}

func (b *fakeProvisionEndpointBuilder) WithKubeconfigBuilder(kubeconfigBuilder kubeconfig.KcBuilder) *fakeProvisionEndpointBuilder {
	b.kcBuilder = kubeconfigBuilder
	return b
}

func (b *fakeProvisionEndpointBuilder) WithConfigurationProvider(provider ConfigurationProvider) *fakeProvisionEndpointBuilder {
	b.providerSpec = provider
	return b
}

func (b *fakeProvisionEndpointBuilder) WithValuesProvider(provider ValuesProvider) *fakeProvisionEndpointBuilder {
	b.valuesProvider = provider
	return b
}

func (b *fakeProvisionEndpointBuilder) WithFreemiumWhitelist(whitelist whitelist.Set) *fakeProvisionEndpointBuilder {
	b.freemiumWhitelist = whitelist
	return b
}

func (b *fakeProvisionEndpointBuilder) WithGvisorWhitelist(wl whitelist.Set) *fakeProvisionEndpointBuilder {
	b.gvisorWhitelist = wl
	return b
}

func (b *fakeProvisionEndpointBuilder) WithPlansConfig(plansConfig PlansConfig) *fakeProvisionEndpointBuilder {
	b.plansConfig = plansConfig
	return b
}

func (b *fakeProvisionEndpointBuilder) WithConfigMapConfigProvider(provider config.ConfigMapConfigProvider) *fakeProvisionEndpointBuilder {
	b.providerConfigProvider = provider
	return b
}

func (b *fakeProvisionEndpointBuilder) WithQuotaClient(client QuotaClient) *fakeProvisionEndpointBuilder {
	b.quotaClient = client
	return b
}

func (b *fakeProvisionEndpointBuilder) WithQuotaWhitelist(whitelist whitelist.Set) *fakeProvisionEndpointBuilder {
	b.quotaWhitelist = whitelist
	return b
}

func (b *fakeProvisionEndpointBuilder) WithRulesService(service *rules.RulesService) *fakeProvisionEndpointBuilder {
	b.rulesService = service
	return b
}

func (b *fakeProvisionEndpointBuilder) WithGardenerClient(client *gardener.Client) *fakeProvisionEndpointBuilder {
	b.gardenerClient = client
	return b
}

func (b *fakeProvisionEndpointBuilder) WithAwsClientFactory(factory aws.ClientFactory) *fakeProvisionEndpointBuilder {
	b.awsClientFactory = factory
	return b
}

func (b *fakeProvisionEndpointBuilder) Build() *ProvisionEndpoint {
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
		b.gvisorWhitelist,
		b.schemaService,
		b.providerSpec,
		b.valuesProvider,
		b.providerConfigProvider,
		b.quotaClient,
		b.quotaWhitelist,
		b.rulesService,
		b.gardenerClient,
		b.awsClientFactory,
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
	provisionEndpoint := NewFakeProvisionEndpointBuilder().
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
	provisionEndpoint := NewFakeProvisionEndpointBuilder().
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
