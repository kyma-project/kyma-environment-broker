package metricsv2

import (
	"context"
	"fmt"
	`sync`
	"time"
	
	"github.com/kyma-project/kyma-environment-broker/internal"
	`github.com/kyma-project/kyma-environment-broker/internal/process`
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// Retention is the default time and date for obtaining operations by the database query
// For performance reasons, it is not possible to query entire operations database table,
// so instead KEB queries the database for last 14 days worth of data and then for deltas
// during the ellapsed time
const (
	Retention       = 14 * 24 * time.Hour
	PollingInterval = 30 * time.Second
)

type operationsGetter interface {
	ListOperationsInTimeRange(from, to time.Time) ([]internal.Operation, error)
}

type operationsCollector struct {
	logger     logrus.FieldLogger
	operations *prometheus.GaugeVec
	lastUpdate time.Time
	db         operationsGetter
	cache      map[string]internal.Operation
	name 	 string
	mu 	   sync.Mutex
}

var _ Exposer = (*operationsCollector)(nil)

// NewOperationsCollectorV2 creates service for exposing prometheus metrics for operations.
//
// This is intended as a replacement for OperationResultCollector to address shortcomings
// of the initial implementation - lack of consistency and non-aggregatable metric desing.
// The underlying data is fetched asynchronously from the KEB SQL database to provide
// consistency and the operation result state is exposed as a label instead of a value to
// enable common gauge aggregation.

// kcp_keb_operation_result


func NewOperationsCollectorV2(ctx context.Context, db operationsGetter, logger logrus.FieldLogger, name string) *operationsCollector {
	svc := &operationsCollector{
		db:         db,
		lastUpdate: time.Now().Add(-Retention),
		logger:     logger,
		cache:      make(map[string]internal.Operation),
		operations: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      name,
			Help:      "Results of operations",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state", "error_category", "error_reason"}),
		name: name,
	}
	go svc.Job(ctx)
	return svc
}

func (s *operationsCollector) setOperation(op internal.Operation, val float64) {
	labels := make(map[string]string)
	labels["operation_id"] = op.ID
	labels["instance_id"] = op.InstanceID
	labels["global_account_id"] = op.GlobalAccountID
	labels["plan_id"] = op.Plan
	labels["type"] = string(op.Type)
	labels["state"] = string(op.State)
	labels["error_category"] = string(op.LastError.Component())
	labels["error_reason"] = string(op.LastError.Reason())
	s.operations.With(labels).Set(val)
}

func (s *operationsCollector) updateOperation(op internal.Operation) {
	oldOp, found := s.cache[op.ID]
	if found {
		s.setOperation(oldOp, 0)
	}
	s.setOperation(op, 1)
	if op.State == domain.Failed || op.State == domain.Succeeded {
		delete(s.cache, op.ID)
	} else {
		s.cache[op.ID] = op
	}
}

func (s *operationsCollector) updateMetrics() (err error) {
	defer func() {
		if r := recover(); r != nil {
			// it's not desirable to panic metrics goroutine, instead it should return and log the error
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()
	now := time.Now()
	operations, err := s.db.ListOperationsInTimeRange(s.lastUpdate, now)
	if err != nil {
		return fmt.Errorf("failed to list operations: %v", err)
	}
	s.logger.Infof("updating operations metrics for: %v operations", len(operations))
	for _, op := range operations {
		s.updateOperation(op)
	}
	s.lastUpdate = now
	return nil
}

func (s *operationsCollector) Handler(ctx context.Context, event interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	switch ev := event.(type) {
	case process.DeprovisioningSucceeded:
		s.updateOperation(ev.Operation.Operation)
	default:
		s.logger.Errorf("unexpected event type: %T", event)
	}
	return nil
}

func (s *operationsCollector) Job(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if err := s.updateMetrics(); err != nil {
		s.logger.Error("failed to update operations metrics", err)
	}
	ticker := time.NewTicker(PollingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.updateMetrics(); err != nil {
				s.logger.Error("failed to update operations metrics", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
