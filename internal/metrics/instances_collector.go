package metrics

import (
	"fmt"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/prometheus/client_golang/prometheus"
)

//
// COPY OF THE internal/metrics/instances_collector.go for test purposes, will be refactored
//

// InstancesStatsGetter provides number of all instances failed, succeeded or orphaned
//
//	(instance exists but the cluster was removed manually from the gardener):
//
// - kcp_keb_instances_total - total number of all instances
// - kcp_keb_global_account_id_instances_total - total number of all instances per global account
// - kcp_keb_ers_context_license_type_total - count of instances grouped by license types
// - kcp_keb_sub_account_id_instances_total - total number of instances per subaccount
type InstancesStatsGetter interface {
	GetActiveInstanceStats() (internal.InstanceStats, error)
	GetERSContextStats() (internal.ERSContextStats, error)
}

type InstancesCollector struct {
	statsGetter InstancesStatsGetter

	instancesDesc        *prometheus.Desc
	instancesPerGAIDDesc *prometheus.Desc
	instancesPerSAIDDesc *prometheus.Desc
	licenseTypeDesc      *prometheus.Desc
	logger               *slog.Logger
}

func NewInstancesCollector(statsGetter InstancesStatsGetter, logger *slog.Logger) *InstancesCollector {
	return &InstancesCollector{
		statsGetter: statsGetter,
		logger:      logger,
		instancesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespaceV2, prometheusSubsystemV2, "instances_total"),
			"The total number of instances",
			[]string{},
			nil),
		instancesPerGAIDDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespaceV2, prometheusSubsystemV2, "global_account_id_instances_total"),
			"The total number of instances by Global Account ID",
			[]string{"global_account_id"},
			nil),
		instancesPerSAIDDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespaceV2, prometheusSubsystemV2, "sub_account_id_instances_total"),
			"The total number of instances by SubAccount ID",
			[]string{"sub_account_id"},
			nil),
		licenseTypeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespaceV2, prometheusSubsystemV2, "ers_context_license_type_total"),
			"count of instances grouped by license types",
			[]string{"license_type"},
			nil),
	}
}

func (c *InstancesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.instancesDesc
	ch <- c.instancesPerGAIDDesc
	ch <- c.instancesPerSAIDDesc
	ch <- c.licenseTypeDesc
}

// Collect implements the prometheus.Collector interface.
func (c *InstancesCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.statsGetter.GetActiveInstanceStats()
	if err != nil {
		c.logger.Error(err.Error())
	} else {
		collect(ch, c.instancesDesc, stats.TotalNumberOfInstances)

		for globalAccountID, num := range stats.PerGlobalAccountID {
			collect(ch, c.instancesPerGAIDDesc, num, globalAccountID)
		}

		for subAccountID, num := range stats.PerSubAcocuntID {
			collect(ch, c.instancesPerSAIDDesc, num, subAccountID)
		}
	}

	stats2, err := c.statsGetter.GetERSContextStats()
	if err != nil {
		c.logger.Error(err.Error())
		return
	}
	for t, num := range stats2.LicenseType {
		collect(ch, c.licenseTypeDesc, num, t)
	}
}

func collect(ch chan<- prometheus.Metric, desc *prometheus.Desc, value int, labelValues ...string) {
	m, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		float64(value),
		labelValues...)

	if err != nil {
		slog.Error(fmt.Sprintf("unable to register metric %s", err.Error()))
		return
	}
	ch <- m
}
