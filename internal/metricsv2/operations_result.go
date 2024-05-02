package metricsv2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

type operationsResult struct {
	logger                           logrus.FieldLogger
	metrics                          *prometheus.GaugeVec
	lastUpdate                       time.Time
	operations                       storage.Operations
	cache                            map[string]internal.Operation
	poolingInterval                  time.Duration
	sync                             sync.Mutex
	finishedOperationRetentionPeriod time.Duration // zero means metrics are stored forever, otherwise they are deleted after this period (starting from the time of operation finish)
}

var _ Exposer = (*operationsResult)(nil)

func NewOperationResult(ctx context.Context, db storage.Operations, cfg Config, logger logrus.FieldLogger) *operationsResult {
	opInfo := &operationsResult{
		operations: db,
		lastUpdate: time.Now().Add(-cfg.OperationResultRetentionPeriod),
		logger:     logger,
		cache:      make(map[string]internal.Operation),
		metrics: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespacev2,
			Subsystem: prometheusSubsystemv2,
			Name:      "operation_result",
			Help:      "Results of metrics",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state", "error_category", "error_reason", "error"}),
		poolingInterval:                  cfg.OperationResultPoolingInterval,
		finishedOperationRetentionPeriod: cfg.OperationResultFinishedOperationRetentionPeriod,
	}
	go opInfo.Job(ctx)
	return opInfo
}

func (s *operationsResult) setOperation(op internal.Operation, val float64) {
	labels := make(map[string]string)
	labels["operation_id"] = op.ID
	labels["instance_id"] = op.InstanceID
	labels["global_account_id"] = op.GlobalAccountID
	labels["plan_id"] = op.ProvisioningParameters.PlanID
	labels["type"] = string(op.Type)
	labels["state"] = string(op.State)
	labels["error_category"] = string(op.LastError.Component())
	labels["error_reason"] = string(op.LastError.Reason())
	labels["error"] = op.LastError.Error()
	s.metrics.With(labels).Set(val)
}

// operation_result metrics works on 0/1 system.
// each metric have labels which identify the operation data by Operation ID
// if metrics with OpId is set to 1, then it means that this event happen in KEB system and will be persisted in Prometheus Server
// metrics set to 0 means that this event is outdated, and will be replaced by new one which happen
func (s *operationsResult) updateOperation(op internal.Operation) {
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
				s.logger.Debugf("Deleted %d metrics for operation %s", count, id)
			}(op.ID)
		}
	} else {
		s.cache[op.ID] = op
	}
}

func (s *operationsResult) updateMetrics() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	now := time.Now()

	operations, err := s.operations.ListOperationsInTimeRange(s.lastUpdate, now)
	s.logger.Debug("UpdateMetrics: %d operations found", len(operations))

	if err != nil {
		return fmt.Errorf("failed to list metrics: %v", err)
	}
	for _, op := range operations {
		s.updateOperation(op)
	}
	s.lastUpdate = now
	return nil
}

func (s *operationsResult) Handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Errorf("panic recovered while handling operation finished event: %v", recovery)
		}
	}()

	switch ev := event.(type) {
	case process.OperationFinished:
		s.logger.Debug("Handling OperationFinished event: OpID=%s State=%s", ev.Operation.ID, ev.Operation.State)
		s.updateOperation(ev.Operation)
	default:
		s.logger.Errorf("Handling OperationFinished, unexpected event type: %T", event)
	}

	return nil
}

func (s *operationsResult) Job(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Errorf("panic recovered while performing operation info job: %v", recovery)
		}
	}()

	if err := s.updateMetrics(); err != nil {
		s.logger.Error("failed to update metrics metrics", err)
	}

	ticker := time.NewTicker(s.poolingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.updateMetrics(); err != nil {
				s.logger.Error("failed to update operation info metrics", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
