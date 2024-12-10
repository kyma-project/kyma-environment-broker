package btpmgrcreds

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron"
)

type Job struct {
	btpOperatorManager *Manager
	logs               *slog.Logger
}

func NewJob(manager *Manager, logs *slog.Logger) *Job {
	return &Job{
		btpOperatorManager: manager,
		logs:               logs,
	}
}

func (s *Job) Start(autoReconcileInterval int, jobReconciliationDelay time.Duration) {
	scheduler := gocron.NewScheduler(time.UTC)
	_, schedulerErr := scheduler.Every(autoReconcileInterval).Minutes().Do(func() {
		s.logs.Info(fmt.Sprintf("runtime-reconciler: scheduled call started at %s", time.Now()))
		_, _, _, _, reconcileErr := s.btpOperatorManager.ReconcileAll(jobReconciliationDelay)
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
