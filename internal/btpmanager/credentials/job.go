package btpmgrcreds

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
)

type Job struct {
	btpOperatorManager *Manager
	logs               *slog.Logger
	metricsRegistry    *prometheus.Registry
	appName            string
}

func NewJob(manager *Manager, logs *slog.Logger, metricsRegistry *prometheus.Registry, appName string) *Job {
	return &Job{
		btpOperatorManager: manager,
		logs:               logs,
		metricsRegistry:    metricsRegistry,
		appName:            appName,
	}
}

func (s *Job) Start(autoReconcileInterval int, jobReconciliationDelay time.Duration) {
	metrics := NewMetrics(s.metricsRegistry, s.appName)
	promHandler := promhttp.HandlerFor(s.metricsRegistry, promhttp.HandlerOpts{Registry: s.metricsRegistry})
	http.Handle("/metrics", promHandler)

	scheduler := gocron.NewScheduler(time.UTC)
	_, schedulerErr := scheduler.Every(autoReconcileInterval).Minutes().Do(func() {
		s.logs.Info(fmt.Sprintf("runtime-reconciler: scheduled call started at %s", time.Now()))
		_, _, _, _, _, reconcileErr := s.btpOperatorManager.ReconcileAll(jobReconciliationDelay, metrics)
		if reconcileErr != nil {
			s.logs.Error(fmt.Sprintf("runtime-reconciler: scheduled call finished with error: %s", reconcileErr))
		} else {
			s.logs.Info(fmt.Sprintf("runtime-reconciler: scheduled call finished with success at %s", time.Now().String()))
		}
	})

	if schedulerErr != nil {
		s.logs.Error(fmt.Sprintf("runtime-reconciler: scheduler failure: %s", schedulerErr))
	}

	s.logs.Info("runtime-listener: start scheduler")
	scheduler.StartAsync()
}
