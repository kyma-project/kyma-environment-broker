package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CredentialsBindingsStatsGetter interface {
	GetCredentialsBindingStats() (internal.CredentialsBindingStats, error)
}

// CredentialsBindingsCollector provides a gauge describing hyperscaler account usage derived from KEB's database:
//   - kcp_keb_v2_instances_per_credentials_binding{credentials_binding,global_account_id} - number of active instances per CredentialsBinding
type CredentialsBindingsCollector struct {
	statsGetter     CredentialsBindingsStatsGetter
	pollingInterval time.Duration
	logger          *slog.Logger

	mu sync.Mutex

	instancesPerCredentialsBinding *prometheus.GaugeVec
}

func NewCredentialsBindingsCollector(statsGetter CredentialsBindingsStatsGetter, pollingInterval time.Duration, logger *slog.Logger) *CredentialsBindingsCollector {
	return &CredentialsBindingsCollector{
		statsGetter:     statsGetter,
		pollingInterval: pollingInterval,
		logger:          logger,
		instancesPerCredentialsBinding: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "instances_per_credentials_binding",
			Help:      "The number of active instances per CredentialsBinding",
		}, []string{"credentials_binding", "global_account_id"}),
	}
}

func (c *CredentialsBindingsCollector) StartCollector(ctx context.Context) {
	go func() {
		c.updateMetrics()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.pollingInterval):
				c.updateMetrics()
			}
		}
	}()
}

func (c *CredentialsBindingsCollector) updateMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats, err := c.statsGetter.GetCredentialsBindingStats()
	if err != nil {
		c.logger.Error(fmt.Sprintf("%s -> failed to get credentials binding stats: %s", logPrefix, err.Error()))
		return
	}

	c.instancesPerCredentialsBinding.Reset()
	for bindingName, count := range stats.InstancesPerCredentialsBinding {
		c.instancesPerCredentialsBinding.With(prometheus.Labels{"credentials_binding": bindingName, "global_account_id": stats.CredentialsBindingToGA[bindingName]}).Set(float64(count))
	}
}
