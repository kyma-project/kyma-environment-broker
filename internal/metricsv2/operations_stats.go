package metricsv2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/setup"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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
	prometheusNamespace = "keb"
	prometheusSubsystem = "kcp_v2"
	metricNamePattern   = "operations_%s_%s_total"
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
		broker.PreviewPlanName,
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

type operationStats struct {
	logger          logrus.FieldLogger
	operations      storage.Operations
	gauges          map[metricKey]prometheus.Gauge
	counters        map[metricKey]prometheus.Counter
	poolingInterval time.Duration
}

func NewOperationsCounters(operations storage.Operations, poolingInterval time.Duration, logger logrus.FieldLogger) *operationStats {
	return &operationStats{
		logger:          logger.WithField("source", "@metricsv2"),
		gauges:          make(map[metricKey]prometheus.Gauge, len(plans)*len(opTypes)*1),
		counters:        make(map[metricKey]prometheus.Counter, len(plans)*len(opTypes)*2),
		operations:      operations,
		poolingInterval: poolingInterval,
	}
}

func (s *operationStats) MustRegister(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Errorf("panic recovered while creating and registering operations metrics: %v", recovery)
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

	go s.statsFromDB(ctx)
}

func (s *operationStats) Handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Error("panic recovered while handling operation counting event: %v", recovery)
		}
	}()

	payload, ok := event.(process.OperationCounting)
	if !ok {
		return fmt.Errorf("expected process.OperationStepProcessed but got %+v", event)
	}

	opState := payload.OpState

	if opState != domain.Failed && opState != domain.Succeeded {
		return fmt.Errorf("operation state is %s, but operation counter support events only from failed or succeded operations", payload.OpState)
	}

	key, err := s.makeKey(payload.OpType, opState, payload.PlanID)
	if err != nil {
		s.logger.Error(err)
		return err
	}

	metric, found := s.counters[key]
	if !found || metric == nil {
		return fmt.Errorf("metric not found for key %s", key)
	}
	s.counters[key].Inc()

	return nil
}

func (s *operationStats) statsFromDB(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			s.logger.Errorf("panic recovered while handling in progress operation counter: %v", recovery)
		}
	}()

	ticker := time.NewTicker(s.poolingInterval)
	for {
		select {
		case <-ticker.C:
			stats, err := s.operations.GetOperationStatsByPlanV2()
			if err != nil {
				s.logger.Errorf("cannot fetch in progress operations from db : %s", err.Error())
				continue
			}
			setStats := make(map[metricKey]struct{})
			for _, stat := range stats {
				key, err := s.makeKey(stat.Type, stat.State, broker.PlanID(stat.PlanID))
				if err != nil {
					s.logger.Error(err)
					continue
				}

				metric, found := s.gauges[key]
				if !found || metric == nil {
					s.logger.Errorf("metric not found for key %s", key)
					continue
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

		case <-ctx.Done():
			return
		}
	}
}

func (s *operationStats) buildName(opType internal.OperationType, opState domain.LastOperationState) (string, error) {
	fmtState := formatOpState(opState)
	fmtType := formatOpType(opType)

	if fmtType == "" || fmtState == "" {
		return "", fmt.Errorf("cannot build name for operation: type: %s, state: %s", opType, opState)
	}

	return prometheus.BuildFQName(
		prometheusNamespace,
		prometheusSubsystem,
		fmt.Sprintf(metricNamePattern, fmtType, fmtState),
	), nil
}

func (s *operationStats) makeKey(opType internal.OperationType, opState domain.LastOperationState, plan broker.PlanID) (metricKey, error) {
	fmtState := formatOpState(opState)
	fmtType := formatOpType(opType)
	if fmtType == "" || fmtState == "" || plan == "" {
		return "", fmt.Errorf("cannot build key for operation: type: %s, state: %s with planID: %s", opType, opState, plan)
	}
	return metricKey(fmt.Sprintf("%s_%s_%s", fmtType, fmtState, plan)), nil
}

func formatOpType(opType internal.OperationType) string {
	if opType == "" {
		return ""
	}
	return string(opType + "ing")
}

func formatOpState(opState domain.LastOperationState) string {
	return strings.ReplaceAll(string(opState), " ", "_")
}
