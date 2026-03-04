package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CredentialsBindingsStatsGetter interface {
	GetCredentialsBindingStats() (internal.CredentialsBindingStats, error)
}

type GardenerCredentialsBindingsLister interface {
	GetCredentialsBindings(labelSelector string) (*unstructured.UnstructuredList, error)
}

// CredentialsBindingsCollector provides gauges describing hyperscaler account usage:
//
//   - kcp_keb_v2_instances_per_credentials_binding{credentials_binding,global_account_id}
//     Number of active instances per CredentialsBinding.
//
//   - kcp_keb_v2_available_credentials_bindings{hyperscaler_type}
//     Number of unclaimed CredentialsBindings per hyperscaler type.
type CredentialsBindingsCollector struct {
	statsGetter             CredentialsBindingsStatsGetter
	gardenerClient          GardenerCredentialsBindingsLister
	dbPollingInterval       time.Duration
	gardenerPollingInterval time.Duration
	logger                  *slog.Logger

	dbMu       sync.Mutex
	gardenerMu sync.Mutex

	instancesPerCredentialsBinding *prometheus.GaugeVec
	availableCredentialsBindings   *prometheus.GaugeVec
}

func NewCredentialsBindingsCollector(
	statsGetter CredentialsBindingsStatsGetter,
	gardenerClient GardenerCredentialsBindingsLister,
	dbPollingInterval time.Duration,
	gardenerPollingInterval time.Duration,
	logger *slog.Logger,
) *CredentialsBindingsCollector {
	return &CredentialsBindingsCollector{
		statsGetter:             statsGetter,
		gardenerClient:          gardenerClient,
		dbPollingInterval:       dbPollingInterval,
		gardenerPollingInterval: gardenerPollingInterval,
		logger:                  logger,
		instancesPerCredentialsBinding: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "instances_per_credentials_binding",
			Help:      "The number of active instances per CredentialsBinding",
		}, []string{"credentials_binding", "global_account_id"}),
		availableCredentialsBindings: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "available_credentials_bindings",
			Help:      "The number of unclaimed CredentialsBindings per hyperscaler type",
		}, []string{"hyperscaler_type"}),
	}
}

func (c *CredentialsBindingsCollector) StartCollector(ctx context.Context) {
	go func() {
		c.updateInstancesMetrics()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.dbPollingInterval):
				c.updateInstancesMetrics()
			}
		}
	}()
	go func() {
		c.updateAvailableMetrics()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.gardenerPollingInterval):
				c.updateAvailableMetrics()
			}
		}
	}()
}

func (c *CredentialsBindingsCollector) updateInstancesMetrics() {
	c.dbMu.Lock()
	defer c.dbMu.Unlock()

	stats, err := c.statsGetter.GetCredentialsBindingStats()
	if err != nil {
		c.logger.Error(fmt.Sprintf("%s -> failed to get credentials binding stats: %s", logPrefix, err.Error()))
		return
	}

	c.instancesPerCredentialsBinding.Reset()
	for bindingName, count := range stats.InstancesPerCredentialsBinding {
		c.instancesPerCredentialsBinding.With(prometheus.Labels{
			"credentials_binding": bindingName,
			"global_account_id":   stats.CredentialsBindingToGA[bindingName],
		}).Set(float64(count))
	}
}

func (c *CredentialsBindingsCollector) updateAvailableMetrics() {
	c.gardenerMu.Lock()
	defer c.gardenerMu.Unlock()

	list, err := c.gardenerClient.GetCredentialsBindings(fmt.Sprintf("!%s", gardener.TenantNameLabelKey))
	if err != nil {
		c.logger.Error(fmt.Sprintf("%s -> failed to get available credentials bindings: %s", logPrefix, err.Error()))
		return
	}

	countByType := make(map[string]int)
	for _, item := range list.Items {
		hyperscalerType := item.GetLabels()[gardener.HyperscalerTypeLabelKey]
		if hyperscalerType == "" {
			continue
		}
		countByType[hyperscalerType]++
	}

	c.availableCredentialsBindings.Reset()
	for hyperscalerType, count := range countByType {
		c.availableCredentialsBindings.With(prometheus.Labels{"hyperscaler_type": hyperscalerType}).Set(float64(count))
	}
}
