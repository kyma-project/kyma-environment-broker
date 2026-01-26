package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/prometheus/client_golang/prometheus"
)

// exposed metrics:
// - kcp_keb_v2_operations_provisioning_failed_total
// - kcp_keb_v2_operations_provisioning_in_progress_total
// - kcp_keb_v2_operations_provisioning_succeeded_total
// - kcp_keb_v2_operations_deprovisioning_failed_total
// - kcp_keb_v2_operations_deprovisioning_in_progress_total
// - kcp_keb_v2_operations_deprovisioning_succeeded_total
// - kcp_keb_v2_operations_update_failed_total
// - kcp_keb_v2_operations_update_in_progress_total
// - kcp_keb_v2_operations_update_succeeded_total

const (
	OpStatsMetricNameTemplate = "operations_%s_%s_total"
	CountersPerPlanType       = 2 // succeeded, failed
	GaugesPerPlanType         = 1 // in_progress
)

var (
	plans   = broker.AvailablePlans.GetAllPlanIDs()
	opTypes = []internal.OperationType{
		internal.OperationTypeProvision,
		internal.OperationTypeDeprovision,
		internal.OperationTypeUpdate,
	}
	opStates = []domain.LastOperationState{
		domain.Failed,
		domain.InProgress,
		domain.Succeeded,
	}
)

type metricKey string

type operationsStats struct {
	logger          *slog.Logger
	operations      storage.Operations
	gauges          map[metricKey]prometheus.Gauge
	counters        map[metricKey]prometheus.Counter
	poolingInterval time.Duration
	sync            sync.Mutex
}

var _ Exposer = (*operationsStats)(nil)

func NewOperationsStats(operations storage.Operations, cfg Config, logger *slog.Logger) *operationsStats {
	return &operationsStats{
		logger:          logger,
		gauges:          make(map[metricKey]prometheus.Gauge, len(plans)*len(opTypes)*GaugesPerPlanType),
		counters:        make(map[metricKey]prometheus.Counter, len(plans)*len(opTypes)*CountersPerPlanType),
		operations:      operations,
		poolingInterval: cfg.OperationStatsPollingInterval,
	}
}

func (s *operationsStats) StartCollector(ctx context.Context) {
	s.logger.Info("Starting operations statistics collector")
	go s.runJob(ctx)
}

func (s *operationsStats) MustRegister() {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while creating and registering operations metrics: %v", recovery))
		}
	}()

	for _, plan := range plans {
		for _, opType := range opTypes {
			labels := prometheus.Labels{"plan_id": string(plan)}

			keyInProgress := s.makeKey(opType, domain.InProgress, plan)
			nameInProgress := s.buildFQName(opType, domain.InProgress)
			s.gauges[keyInProgress] = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name:        nameInProgress,
					ConstLabels: labels,
				},
			)
			prometheus.MustRegister(s.gauges[keyInProgress])

			keySucceeded := s.makeKey(opType, domain.Succeeded, plan)
			nameSucceeded := s.buildFQName(opType, domain.Succeeded)
			s.counters[keySucceeded] = prometheus.NewCounter(
				prometheus.CounterOpts{
					Name:        nameSucceeded,
					ConstLabels: labels,
				},
			)
			prometheus.MustRegister(s.counters[keySucceeded])

			keyFailed := s.makeKey(opType, domain.Failed, plan)
			nameFailed := s.buildFQName(opType, domain.Failed)
			s.counters[keyFailed] = prometheus.NewCounter(
				prometheus.CounterOpts{
					Name:        nameFailed,
					ConstLabels: labels,
				},
			)
			prometheus.MustRegister(s.counters[keyFailed])
		}
	}
}

func (s *operationsStats) UpdateMetrics(_ context.Context, event interface{}) error {
	defer s.sync.Unlock()
	s.sync.Lock()

	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while handling operation counting event: %v", recovery))
		}
	}()

	payload, ok := event.(process.OperationFinished)
	if !ok {
		return fmt.Errorf("expected process.OperationFinished but got %+v", event)
	}

	opState := payload.Operation.State

	if opState != domain.Failed && opState != domain.Succeeded {
		return fmt.Errorf("operation state is %s, but operation counter supports only failed or succeeded operations events ", payload.Operation.State)
	}

	if payload.PlanID == "" {
		return fmt.Errorf("plan ID is empty in operation finished event for operation ID %s", payload.Operation.ID)
	}

	if payload.Operation.Type == "" {
		return fmt.Errorf("operation type is empty in operation finished event for operation ID %s", payload.Operation.ID)
	}

	key := s.makeKey(payload.Operation.Type, opState, broker.PlanIDType(payload.PlanID))

	metric, found := s.counters[key]
	if !found || metric == nil {
		return fmt.Errorf("metric not found for key %s, unable to increment", key)
	}
	s.counters[key].Inc()

	return nil
}

func (s *operationsStats) runJob(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while handling in progress operation counter: %v", recovery))
		}
	}()

	fmt.Printf("starting operations stats metrics runJob with interval %s\n", s.poolingInterval)
	if err := s.UpdateGauges(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to update metrics gauges initially: %v", err))
	}

	ticker := time.NewTicker(s.poolingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.UpdateGauges(); err != nil {
				s.logger.Error(fmt.Sprintf("failed to update operation metrics gauges: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *operationsStats) UpdateGauges() error {
	defer s.sync.Unlock()
	s.sync.Lock()

	statsFromDB, err := s.operations.GetOperationStatsByPlanV2()
	if err != nil {
		return fmt.Errorf("cannot fetch operations statistics by plan from operations table : %s", err.Error())
	}
	statsSet := make(map[metricKey]struct{})
	for _, stat := range statsFromDB {
		key := s.makeKey(stat.Type, stat.State, broker.PlanIDType(stat.PlanID))

		metric, found := s.gauges[key]
		if !found || metric == nil {
			return fmt.Errorf("metric not found for key %s", key)
		}
		metric.Set(float64(stat.Count))
		statsSet[key] = struct{}{}
	}

	for key, metric := range s.gauges {
		if _, ok := statsSet[key]; ok {
			continue
		}
		metric.Set(0)
	}
	return nil
}

func (s *operationsStats) buildFQName(opType internal.OperationType, opState domain.LastOperationState) string {
	return prometheus.BuildFQName(prometheusNamespaceV2, prometheusSubsystemV2, fmt.Sprintf(OpStatsMetricNameTemplate, formatOpType(opType), formatOpState(opState)))
}

func (s *operationsStats) GetCounter(opType internal.OperationType, opState domain.LastOperationState, planID broker.PlanIDType) prometheus.Counter {
	key := s.makeKey(opType, opState, planID)
	s.sync.Lock()
	defer s.sync.Unlock()
	return s.counters[key]
}

func (s *operationsStats) makeKey(opType internal.OperationType, opState domain.LastOperationState, planID broker.PlanIDType) metricKey {
	return metricKey(fmt.Sprintf("%s_%s_%s", formatOpType(opType), formatOpState(opState), string(planID)))
}

func formatOpType(opType internal.OperationType) string {
	switch opType {
	case internal.OperationTypeProvision, internal.OperationTypeDeprovision:
		return string(opType + "ing")
	case internal.OperationTypeUpdate:
		return "updating"
	default:
		return ""
	}
}

func formatOpState(opState domain.LastOperationState) string {
	return strings.ReplaceAll(string(opState), " ", "_")
}
