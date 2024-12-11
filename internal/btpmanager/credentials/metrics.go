package btpmgrcreds

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	skippedSecrets *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer, namespace string) *Metrics {
	m := &Metrics{
		skippedSecrets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "reconciled_secrets",
			Help:      "Reconciled secrets.",
		}, []string{"shoot", "state"}),
	}
	reg.MustRegister(m.skippedSecrets)
	return m
}
