package metricsv2

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	prometheusNamespacev2 = "kcp"
	prometheusSubsystemv2 = "keb_v2"
	logPrefix             = "@metricsv2"
)

// Exposer gathers metrics and keeps these in memory and exposes to prometheus for fetching, it gathers them by:
// listening in real time for events by "Handler"
// fetching data from database by "Job"

type Exposer interface {
	Handler(ctx context.Context, event interface{}) error
	Job(ctx context.Context)
}

type Config struct {
	Enabled                                         bool          `envconfig:"default=false"`
	OperationResultRetentionPeriod                  time.Duration `envconfig:"default=1h"`
	OperationResultPollingInterval                  time.Duration `envconfig:"default=1m"`
	OperationStatsPollingInterval                   time.Duration `envconfig:"default=1m"`
	OperationResultFinishedOperationRetentionPeriod time.Duration `envconfig:"default=3h"`
	BindingsStatsPollingInterval                    time.Duration `envconfig:"default=1m"`
}

type RegisterContainer struct {
	OperationResult            *operationsResult
	OperationStats             *OperationStats
	OperationDurationCollector *OperationDurationCollector
	InstancesCollector         *InstancesCollector
}

func Register(ctx context.Context, sub event.Subscriber, db storage.BrokerStorage, cfg Config, logger logrus.FieldLogger) *RegisterContainer {
	logger = logger.WithField("from:", logPrefix)
	logrus.Infof("Registering metricsv2")
	opDurationCollector := NewOperationDurationCollector(logger)
	prometheus.MustRegister(opDurationCollector)

	opInstanceCollector := NewInstancesCollector(db.Instances(), logger)
	prometheus.MustRegister(opInstanceCollector)

	opResult := NewOperationResult(ctx, db.Operations(), cfg, logger)

	opStats := NewOperationsStats(db.Operations(), cfg, logger)
	opStats.MustRegister(ctx)

	bindingStats := NewBindingStatsCollector(db.Bindings(), cfg.BindingsStatsPollingInterval, logger)
	bindingStats.MustRegister(ctx)

	bindDurationCollector := NewBindDurationCollector(logger)
	prometheus.MustRegister(bindDurationCollector)

	bindCrestedCollector := NewBindingCreationCollector()
	prometheus.MustRegister(bindCrestedCollector)

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationFinished{}, opStats.Handler)
	sub.Subscribe(process.OperationFinished{}, opResult.Handler)

	sub.Subscribe(broker.BindRequestProcessed{}, bindDurationCollector.OnBindingExecuted)
	sub.Subscribe(broker.UnbindRequestProcessed{}, bindDurationCollector.OnUnbindingExecuted)
	sub.Subscribe(broker.BindingCreated{}, bindCrestedCollector.OnBindingCreated)

	logger.Infof(fmt.Sprintf("%s -> enabled", logPrefix))

	return &RegisterContainer{
		OperationResult:            opResult,
		OperationStats:             opStats,
		OperationDurationCollector: opDurationCollector,
		InstancesCollector:         opInstanceCollector,
	}
}

func randomState() domain.LastOperationState {
	return opStates[rand.Intn(len(opStates))]
}

func randomType() internal.OperationType {
	return opTypes[rand.Intn(len(opTypes))]
}

func randomPlanId() string {
	return string(plans[rand.Intn(len(plans))])
}

func randomCreatedAt() time.Time {
	return time.Now().UTC().Add(-time.Duration(rand.Intn(60)) * time.Minute)
}

func randomUpdatedAtAfterCreatedAt() time.Time {
	return randomCreatedAt().Add(time.Duration(rand.Intn(10)) * time.Minute)
}

func GetRandom(createdAt time.Time, state domain.LastOperationState) internal.Operation {
	return internal.Operation{
		ID:         uuid.New().String(),
		InstanceID: uuid.New().String(),
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: randomPlanId(),
		},
		CreatedAt: createdAt,
		UpdatedAt: randomUpdatedAtAfterCreatedAt(),
		Type:      randomType(),
		State:     state,
	}
}

func GetLabels(op internal.Operation) map[string]string {
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
	return labels
}
