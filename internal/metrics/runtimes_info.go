package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

// RuntimesInfoRequestsCollector provides a counter that tracks the total number of requests to the runtimes info endpoint:
// - kcp_keb_v2_runtimes_info_requests_total
type RuntimesInfoRequestsCollector struct {
	requestTotal *prometheus.CounterVec
}

func NewRuntimesInfoRequestsCollector() *RuntimesInfoRequestsCollector {
	return &RuntimesInfoRequestsCollector{
		requestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "runtimes_info_requests_total",
			Help:      "Total number of requests to the runtimes info endpoint",
		}, nil),
	}
}

func (c *RuntimesInfoRequestsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.requestTotal.Describe(ch)
}

func (c *RuntimesInfoRequestsCollector) Collect(ch chan<- prometheus.Metric) {
	c.requestTotal.Collect(ch)
}

func (c *RuntimesInfoRequestsCollector) OnRequest(ctx context.Context, ev interface{}) error {
	c.requestTotal.WithLabelValues().Inc()
	return nil
}
