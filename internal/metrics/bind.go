package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
)

// BindDurationCollector provides histograms which describe the time of bind/unbind request processing:
// - kcp_keb_v2_bind_duration_millisecond
// - kcp_keb_v2_unbind_duration_millisecond
type BindDurationCollector struct {
	bindHistogram   *prometheus.HistogramVec
	unbindHistogram *prometheus.HistogramVec
	logger          *slog.Logger
}

func NewBindDurationCollector(logger *slog.Logger) *BindDurationCollector {
	return &BindDurationCollector{
		bindHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "bind_duration_millisecond",
			Help:      "The time of the bind request",
			Buckets:   prometheus.LinearBuckets(50, 200, 15),
		}, []string{}),
		unbindHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "unbind_duration_millisecond",
			Help:      "The time of the unbind request",
			Buckets:   prometheus.LinearBuckets(50, 200, 15),
		}, []string{}),
		logger: logger,
	}
}

func (c *BindDurationCollector) Describe(ch chan<- *prometheus.Desc) {
	c.bindHistogram.Describe(ch)
	c.unbindHistogram.Describe(ch)
}

func (c *BindDurationCollector) Collect(ch chan<- prometheus.Metric) {
	c.bindHistogram.Collect(ch)
	c.unbindHistogram.Collect(ch)
}

func (c *BindDurationCollector) OnBindingExecuted(ctx context.Context, ev interface{}) error {
	obj := ev.(broker.BindRequestProcessed)
	c.bindHistogram.WithLabelValues().Observe(float64(obj.ProcessingDuration.Milliseconds()))
	return nil
}

func (c *BindDurationCollector) OnUnbindingExecuted(ctx context.Context, ev interface{}) error {
	obj := ev.(broker.UnbindRequestProcessed)
	c.unbindHistogram.WithLabelValues().Observe(float64(obj.ProcessingDuration.Milliseconds()))
	return nil
}

type BindingCreationCollector struct {
	bindingCreated *prometheus.CounterVec
}

// BindingCreationCollector provides a counter which shows the total number of created bindings:
// - kcp_keb_v2_binding_created_total{plan_id}
func NewBindingCreationCollector() *BindingCreationCollector {
	return &BindingCreationCollector{
		bindingCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "binding_created_total",
			Help:      "The total number of created bindings",
		}, []string{"plan_id"}),
	}
}

func (c *BindingCreationCollector) Describe(ch chan<- *prometheus.Desc) {
	c.bindingCreated.Describe(ch)
}

func (c *BindingCreationCollector) Collect(ch chan<- prometheus.Metric) {
	c.bindingCreated.Collect(ch)
}

func (c *BindingCreationCollector) OnBindingCreated(ctx context.Context, ev interface{}) error {
	obj := ev.(broker.BindingCreated)
	c.bindingCreated.WithLabelValues(obj.PlanID).Inc()
	return nil
}

type BindingStatitics struct {
	db     storage.Bindings
	logger *slog.Logger

	sync                                 sync.Mutex
	pollingInterval                      time.Duration
	MinutesSinceEarliestExpirationMetric prometheus.Gauge
}

// NewBindingStatsCollector provides a collector which shows the time in minutes since the earliest binding expiration:
// - kcp_keb_v2_minutes_since_earliest_binding_expiration
func NewBindingStatsCollector(db storage.Bindings, pollingInterval time.Duration, logger *slog.Logger) *BindingStatitics {
	return &BindingStatitics{
		db:              db,
		logger:          logger,
		pollingInterval: pollingInterval,
		MinutesSinceEarliestExpirationMetric: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "minutes_since_earliest_binding_expiration",
			Help: "Specifies the time in minutes since the earliest binding expiration. " +
				"The value should not be greater than the binding cleaning runJob interval. The metric is created to detect problems with the runJob.",
		}),
	}
}

func (c *BindingStatitics) MustRegister() {
	prometheus.MustRegister(c.MinutesSinceEarliestExpirationMetric)
}

func (c *BindingStatitics) Job(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			c.logger.Error(fmt.Sprintf("panic recovered while handling in progress operation counter: %v", recovery))
		}
	}()

	if err := c.updateMetrics(); err != nil {
		c.logger.Error(fmt.Sprintf("failed to update metrics: %v", err))
	}

	ticker := time.NewTicker(c.pollingInterval)
	for {
		select {
		case <-ticker.C:
			if err := c.updateMetrics(); err != nil {
				c.logger.Error(fmt.Sprintf("failed to update operation stats metrics: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *BindingStatitics) updateMetrics() error {
	defer c.sync.Unlock()
	c.sync.Lock()

	stats, err := c.db.GetStatistics()
	if err != nil {
		return fmt.Errorf("cannot fetch in progress metrics from operations : %s", err.Error())
	}
	c.MinutesSinceEarliestExpirationMetric.Set(stats.MinutesSinceEarliestExpiration)
	return nil
}

func (c *BindingStatitics) StartCollector(ctx context.Context) {
	go c.Job(ctx)
}
