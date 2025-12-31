package metricsv2

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

const (
	OpStatsMetricNameTemplate = "operations_%s_%s_total"
)

var (
	//TODO: get plans dynamically from broker plans
	plans = []broker.PlanID{
		broker.AzurePlanID,
		broker.AzureLitePlanID,
		broker.AWSPlanID,
		broker.GCPPlanID,
		broker.SapConvergedCloudPlanID,
		broker.TrialPlanID,
		broker.FreemiumPlanID,
		broker.PreviewPlanID,
	}
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
		gauges:          make(map[metricKey]prometheus.Gauge, len(plans)*len(opTypes)*1),   // TODO: get rid of magic number
		counters:        make(map[metricKey]prometheus.Counter, len(plans)*len(opTypes)*2), // TODO: get rid of magic number
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
			for _, opState := range opStates {
				key := s.makeKey(opType, opState, plan)
				name := s.buildFQName(opType, opState)
				labels := prometheus.Labels{"plan_id": string(plan)}
				switch opState {
				case domain.InProgress:
					s.gauges[key] = prometheus.NewGauge(
						prometheus.GaugeOpts{
							Name:        name,
							ConstLabels: labels,
						},
					)
					prometheus.MustRegister(s.gauges[key])
				case domain.Failed, domain.Succeeded:
					s.counters[key] = prometheus.NewCounter(
						prometheus.CounterOpts{
							Name:        name,
							ConstLabels: labels,
						},
					)
					prometheus.MustRegister(s.counters[key])
				}
			}
		}
	}
}

func (s *operationsStats) Handler(_ context.Context, event interface{}) error {
	defer s.sync.Unlock()
	s.sync.Lock()

	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while handling operation counting event: %v", recovery))
		}
	}()

	payload, ok := event.(process.OperationFinished)
	if !ok {
		return fmt.Errorf("expected process.OperationStepProcessed but got %+v", event)
	}

	opState := payload.Operation.State

	if opState != domain.Failed && opState != domain.Succeeded {
		return fmt.Errorf("operation state is %s, but operation counter supports only failed or succeded operations events ", payload.Operation.State)
	}

	if payload.PlanID == "" {
		return fmt.Errorf("plan ID is empty in operation finished event for operation ID %s", payload.Operation.ID)
	}

	if payload.Operation.Type == "" {
		return fmt.Errorf("operation type is empty in operation finished event for operation ID %s", payload.Operation.ID)
	}

	key := s.makeKey(payload.Operation.Type, opState, payload.PlanID)

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
	if err := s.UpdateStats(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to update metrics metrics: %v", err))
	}

	ticker := time.NewTicker(s.poolingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.UpdateStats(); err != nil {
				s.logger.Error(fmt.Sprintf("failed to update operation stats metrics: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *operationsStats) UpdateStats() error {
	defer s.sync.Unlock()
	s.sync.Lock()

	statsFromDB, err := s.operations.GetOperationStatsByPlanV2()
	if err != nil {
		return fmt.Errorf("cannot fetch operations statistics by plan from operations table : %s", err.Error())
	}
	statsSet := make(map[metricKey]struct{})
	for _, stat := range statsFromDB {
		key := s.makeKey(stat.Type, stat.State, broker.PlanID(stat.PlanID))

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
	return prometheus.BuildFQName(prometheusNamespacev2, prometheusSubsystemv2, fmt.Sprintf(OpStatsMetricNameTemplate, formatOpType(opType), formatOpState(opState)))
}

// TODO: is it needed? It is used only in tests
func (s *operationsStats) GetCounter(opType internal.OperationType, opState domain.LastOperationState, plan broker.PlanID) prometheus.Counter {
	key := s.makeKey(opType, opState, plan)
	s.sync.Lock()
	defer s.sync.Unlock()
	return s.counters[key]
}

func (s *operationsStats) makeKey(opType internal.OperationType, opState domain.LastOperationState, plan broker.PlanID) metricKey {
	return metricKey(fmt.Sprintf("%s_%s_%s", formatOpType(opType), formatOpState(opState), plan))
}

func formatOpType(opType internal.OperationType) string {
	switch opType {
	case
		internal.OperationTypeProvision,
		internal.OperationTypeDeprovision,
		internal.OperationTypeUpdate:
		return string(opType + "ing")
	default:
		return ""
	}
}

func formatOpState(opState domain.LastOperationState) string {
	return strings.ReplaceAll(string(opState), " ", "_")
}
