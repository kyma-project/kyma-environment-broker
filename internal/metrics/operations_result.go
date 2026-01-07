package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type operationsResults struct {
	logger                           *slog.Logger
	metrics                          *prometheus.GaugeVec
	lastUpdate                       time.Time
	operations                       storage.Operations
	cache                            map[string]internal.Operation
	pollingInterval                  time.Duration
	sync                             sync.Mutex
	finishedOperationRetentionPeriod time.Duration // zero means metrics are stored forever, otherwise they are deleted after this period (starting from the time of operation finish)
}

var _ Exposer = (*operationsResults)(nil)

func NewOperationsResults(db storage.Operations, cfg Config, logger *slog.Logger) *operationsResults {
	opInfo := &operationsResults{
		operations: db,
		lastUpdate: time.Now().UTC().Add(-cfg.OperationResultRetentionPeriod),
		logger:     logger,
		cache:      make(map[string]internal.Operation),
		metrics: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespacev2,
			Subsystem: prometheusSubsystemv2,
			Name:      "operation_result",
			Help:      "Metrics of operations results",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state", "error_category", "error_reason", "error"}),
		pollingInterval:                  cfg.OperationResultPollingInterval,
		finishedOperationRetentionPeriod: cfg.OperationResultFinishedOperationRetentionPeriod,
	}

	return opInfo
}

func (s *operationsResults) StartCollector(ctx context.Context) {
	s.logger.Info("Starting operations results collector")
	go s.runJob(ctx)
}

func (s *operationsResults) Metrics() *prometheus.GaugeVec {
	return s.metrics
}

func (s *operationsResults) setOperation(op internal.Operation, val float64) {
	labels := GetLabels(op)
	s.metrics.With(labels).Set(val)
}

// operation_result metrics works on 0/1 system.
// each metric have labels which identify the operation data by Operation ID
// if metrics with OpId is set to 1, then it means that this event happen in KEB system and will be persisted in Prometheus Server
// metrics set to 0 means that this event is outdated, and will be replaced by new one
func (s *operationsResults) updateOperation(op internal.Operation) {
	defer s.sync.Unlock()
	s.sync.Lock()

	oldOp, found := s.cache[op.ID]
	if found {
		s.setOperation(oldOp, 0)
	}
	s.setOperation(op, 1)
	if op.State == domain.Failed || op.State == domain.Succeeded {
		delete(s.cache, op.ID)

		// keep those metric and remove after finishedOperationRetentionPeriod
		if s.finishedOperationRetentionPeriod > 0 {
			go func(id string) {
				time.Sleep(s.finishedOperationRetentionPeriod)
				count := s.metrics.DeletePartialMatch(prometheus.Labels{"operation_id": id})
				s.logger.Debug(fmt.Sprintf("Deleted %d metrics for operation %s", count, id))
			}(op.ID)
		}
	} else {
		s.cache[op.ID] = op
	}
}

func (s *operationsResults) UpdateOperationResultsMetrics() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	now := time.Now().UTC()

	operations, err := s.operations.ListOperationsInTimeRange(s.lastUpdate, now)
	if len(operations) != 0 {
		s.logger.Debug(fmt.Sprintf("UpdateStatsMetrics: %d operations found", len(operations)))
	}
	if err != nil {
		return fmt.Errorf("failed to list metrics: %v", err)
	}

	for _, op := range operations {
		s.updateOperation(op)
	}
	s.lastUpdate = now
	return nil
}

func (s *operationsResults) Handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while handling operation finished event: %v", recovery))
		}
	}()

	switch ev := event.(type) {
	case process.OperationFinished:
		s.logger.Debug(fmt.Sprintf("Handling OperationFinished event: OpID=%s State=%s", ev.Operation.ID, ev.Operation.State))
		s.updateOperation(ev.Operation)
	default:
		s.logger.Error(fmt.Sprintf("Handling OperationFinished, unexpected event type: %T", event))
	}

	return nil
}

func (s *operationsResults) runJob(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while collecting operations results metrics: %v", recovery))
		}
	}()

	if err := s.UpdateOperationResultsMetrics(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to update metrics: %v", err))
	}

	ticker := time.NewTicker(s.pollingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.UpdateOperationResultsMetrics(); err != nil {
				s.logger.Error(fmt.Sprintf("failed to update operations info metrics: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}
