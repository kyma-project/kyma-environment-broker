package metricsv2

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/setup"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
)

// exposed metrics:
// - kcp_keb_v2_operations_{plan_name}_provisioning_failed_total
// - kcp_keb_v2_operations_{plan_name}_provisioning_in_progress_total
// - kcp_keb_v2_operations_{plan_name}_provisioning_succeeded_total
// - kcp_keb_v2_operations_{plan_name}_deprovisioning_failed_total
// - kcp_keb_v2_operations_{plan_name}_deprovisioning_in_progress_total
// - kcp_keb_v2_operations_{plan_name}_deprovisioning_succeeded_total
// - kcp_keb_v2_operations_{plan_name}_update_failed_total
// - kcp_keb_v2_operations_{plan_name}_update_in_progress_total
// - kcp_keb_v2_operations_{plan_name}_update_succeeded_total

const (
	OpStatsMetricName = "operations_%s_%s_total"
)

var (
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

type OperationStats struct {
	logger          *slog.Logger
	operations      storage.Operations
	gauges          map[metricKey]prometheus.Gauge
	counters        map[metricKey]prometheus.Counter
	poolingInterval time.Duration
	sync            sync.Mutex
}

var _ Exposer = (*OperationStats)(nil)

func NewOperationsStats(operations storage.Operations, cfg Config, logger *slog.Logger) *OperationStats {
	return &OperationStats{
		logger:          logger,
		gauges:          make(map[metricKey]prometheus.Gauge, len(plans)*len(opTypes)*1),
		counters:        make(map[metricKey]prometheus.Counter, len(plans)*len(opTypes)*2),
		operations:      operations,
		poolingInterval: cfg.OperationStatsPollingInterval,
	}
}

func (s *OperationStats) MustRegister(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while creating and registering operations metrics: %v", recovery))
		}
	}()

	for _, plan := range plans {
		for _, opType := range opTypes {
			for _, opState := range opStates {
				key, err := s.makeKey(opType, opState, plan)
				setup.FatalOnError(err)
				name, err := s.buildName(opType, opState)
				setup.FatalOnError(err)
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

	go s.Job(ctx)
}

func (s *OperationStats) Handler(_ context.Context, event interface{}) error {
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
		return fmt.Errorf("operation state is %s, but operation counter support events only from failed or succeded operations", payload.Operation.State)
	}

	key, err := s.makeKey(payload.Operation.Type, opState, payload.PlanID)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	metric, found := s.counters[key]
	if !found || metric == nil {
		return fmt.Errorf("metric not found for key %s", key)
	}
	s.counters[key].Inc()

	return nil
}

func (s *OperationStats) Job(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error(fmt.Sprintf("panic recovered while handling in progress operation counter: %v", recovery))
		}
	}()

	if err := s.updateMetrics(); err != nil {
		s.logger.Error(fmt.Sprintf("failed to update metrics metrics: %v", err))
	}

	ticker := time.NewTicker(s.poolingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.updateMetrics(); err != nil {
				s.logger.Error(fmt.Sprintf("failed to update operation stats metrics: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *OperationStats) updateMetrics() error {
	defer s.sync.Unlock()
	s.sync.Lock()

	stats, err := s.operations.GetOperationStatsByPlanV2()
	if err != nil {
		return fmt.Errorf("cannot fetch in progress metrics from operations : %s", err.Error())
	}
	setStats := make(map[metricKey]struct{})
	for _, stat := range stats {
		key, err := s.makeKey(stat.Type, stat.State, broker.PlanID(stat.PlanID))
		if err != nil {
			return err
		}

		metric, found := s.gauges[key]
		if !found || metric == nil {
			return fmt.Errorf("metric not found for key %s", key)
		}
		metric.Set(float64(stat.Count))
		setStats[key] = struct{}{}
	}

	for key, metric := range s.gauges {
		if _, ok := setStats[key]; ok {
			continue
		}
		metric.Set(0)
	}
	return nil
}

func (s *OperationStats) buildName(opType internal.OperationType, opState domain.LastOperationState) (string, error) {
	fmtState := formatOpState(opState)
	fmtType := formatOpType(opType)

	if fmtType == "" || fmtState == "" {
		return "", fmt.Errorf("cannot build name for operation: type: %s, state: %s", opType, opState)
	}

	return prometheus.BuildFQName(
		prometheusNamespacev2,
		prometheusSubsystemv2,
		fmt.Sprintf(OpStatsMetricName, fmtType, fmtState),
	), nil
}

func (s *OperationStats) Metric(opType internal.OperationType, opState domain.LastOperationState, plan broker.PlanID) (prometheus.Counter, error) {
	key, err := s.makeKey(opType, opState, plan)
	if err != nil {
		s.logger.Error(err.Error())
		return prometheus.NewGauge(prometheus.GaugeOpts{}), err
	}
	s.sync.Lock()
	defer s.sync.Unlock()
	return s.counters[key], nil
}

func (s *OperationStats) makeKey(opType internal.OperationType, opState domain.LastOperationState, plan broker.PlanID) (metricKey, error) {
	fmtState := formatOpState(opState)
	fmtType := formatOpType(opType)
	if fmtType == "" || fmtState == "" || plan == "" {
		return "", fmt.Errorf("cannot build key for operation: type - '%s', state - '%s' with planID - '%s'", opType, opState, plan)
	}
	return metricKey(fmt.Sprintf("%s_%s_%s", fmtType, fmtState, plan)), nil
}

func formatOpType(opType internal.OperationType) string {
	switch opType {
	case internal.OperationTypeProvision, internal.OperationTypeDeprovision:
		return string(opType + "ing")
	case internal.OperationTypeUpdate:
		return "updating"
	case internal.OperationTypeUpgradeCluster:
		return "upgrading_cluster"
	default:
		return ""
	}
}

func formatOpState(opState domain.LastOperationState) string {
	return strings.ReplaceAll(string(opState), " ", "_")
}
