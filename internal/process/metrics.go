package process

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// QueueWorkersInUse tracks the number of workers currently processing items for each queue
	QueueWorkersInUse = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kcp",
		Subsystem: "keb_v2",
		Name:      "queue_workers_in_use",
		Help:      "Number of queue workers currently processing items",
	}, []string{"queue_name"})
)

func init() {
	prometheus.MustRegister(QueueWorkersInUse)
}
