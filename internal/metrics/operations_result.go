package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type resultsCollector struct {
	logger                           *slog.Logger
	metrics                          *prometheus.GaugeVec
	lastUpdate                       time.Time
	operations                       storage.Operations
	cache                            map[string]cachedOperationType
	pollingInterval                  time.Duration
	sync                             sync.Mutex
	finishedOperationRetentionPeriod time.Duration // zero means metrics are stored forever; otherwise they are deleted after this period (starting from the time of operation finish)
}

type cachedOperationType struct {
	labels   map[string]string
	deleteAt time.Time
}

var _ Exposer = (*resultsCollector)(nil)

func newResultsCollector(db storage.Operations, cfg Config, logger *slog.Logger) *resultsCollector {
	collector := &resultsCollector{
		operations: db,
		lastUpdate: time.Now().UTC().Add(-cfg.OperationResultRetentionPeriod),
		logger:     logger,
		cache:      make(map[string]cachedOperationType),
		metrics: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespaceV2,
			Subsystem: prometheusSubsystemV2,
			Name:      "operation_result",
			Help:      "Metrics of operations results",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state", "error_category", "error_reason", "error"}),
		pollingInterval:                  cfg.OperationResultPollingInterval,
		finishedOperationRetentionPeriod: cfg.OperationResultFinishedOperationRetentionPeriod,
	}

	return collector
}

func (s *resultsCollector) startCollectorJob(ctx context.Context) {
	s.logger.Info("Starting operation results collector job")
	go s.runJob(ctx)
}

func (s *resultsCollector) deleteMarked(now time.Time) {
	s.sync.Lock()
	defer s.sync.Unlock()
	for id, cachedOp := range s.cache {
		if !cachedOp.deleteAt.IsZero() && cachedOp.deleteAt.Before(now) {
			delete(s.cache, id)
			count := s.metrics.DeletePartialMatch(prometheus.Labels{"operation_id": id})
			s.logger.Debug(fmt.Sprintf("Deleted %d metrics for operation %s", count, id))
		}
	}
}

func (s *resultsCollector) Metrics() *prometheus.GaugeVec {
	return s.metrics
}

// operation_result metrics works on 0/1 system.
// each metric has labels which identify the operation data by Operation ID
// if metrics with OpId is set to 1, then it means that this event happens in a KEB system and will be persisted in Prometheus Server
// metrics set to 0 means that this event is outdated, and will be replaced by a new one
func (s *resultsCollector) updateMetricsForOperation(operation internal.Operation) {
	defer s.sync.Unlock()
	s.sync.Lock()

	cachedOperation, found := s.cache[operation.ID]
	if found {
		if !cachedOperation.deleteAt.IsZero() {
			return
		}
		s.metrics.With(cachedOperation.labels).Set(0)
	}
	// TODO so we calculate labels every time we get the same operation - rethink caching strategy and footprint
	operationLabels := GetLabels(operation)
	s.metrics.With(operationLabels).Set(1)

	if operation.State == domain.Failed || operation.State == domain.Succeeded {
		s.markForDeletion(operation.ID)
	} else {
		s.cache[operation.ID] = cachedOperationType{labels: operationLabels}
	}
}

func (s *resultsCollector) markForDeletion(operationID string) {
	cachedOp := s.cache[operationID]
	cachedOp.deleteAt = time.Now().UTC().Add(s.finishedOperationRetentionPeriod)
	s.cache[operationID] = cachedOp
}

func (s *resultsCollector) UpdateResultMetricsInTimeRange() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	now := time.Now().UTC()

	operations, err := s.operations.ListOperationsInTimeRange(s.lastUpdate, now)
	if len(operations) != 0 {
		s.logger.Debug(fmt.Sprintf("UpdateGauges: %d operations found", len(operations)))
	}
	if err != nil {
		return fmt.Errorf("failed to list metrics: %v", err)
	}

	for _, op := range operations {
		s.updateMetricsForOperation(op)
	}
	s.lastUpdate = now
	return nil
}

func (s *resultsCollector) UpdateMetrics(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while handling operation finished event: %v", recovery))
		}
	}()

	switch ev := event.(type) {
	case process.OperationFinished:
		s.logger.Debug(fmt.Sprintf("Handling OperationFinished event: OpID=%s State=%s", ev.Operation.ID, ev.Operation.State))
		s.updateMetricsForOperation(ev.Operation)
	default:
		s.logger.Error(fmt.Sprintf("Handling OperationFinished, unexpected event type: %T", event))
	}

	return nil
}

func (s *resultsCollector) runJob(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while collecting operations results metrics: %v", recovery))
		}
	}()

	if err := s.UpdateResultMetricsInTimeRange(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to update metrics: %v", err))
	}

	ticker := time.NewTicker(s.pollingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.UpdateResultMetricsInTimeRange(); err != nil {
				s.logger.Error(fmt.Sprintf("failed to update operations info metrics: %v", err))
			}
			if s.finishedOperationRetentionPeriod > 0 {
				s.deleteMarked(time.Now().UTC())
			}
		case <-ctx.Done():
			return
		}
	}
}
