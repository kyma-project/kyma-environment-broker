package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespacev2 = "kcp"
	prometheusSubsystemv2 = "keb_v2"
	logPrefix             = "@metrics"
)

// Exposer gathers metrics and keeps these in memory and exposes to prometheus for fetching, it gathers them by:
// listening in real time for events by "Handler"
// fetching data from database by "runJob"

type Exposer interface {
	Handler(ctx context.Context, event interface{}) error
	runJob(ctx context.Context)
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
	OperationResult            *operationsResults
	OperationStats             *operationsStats
	OperationDurationCollector *OperationDurationCollector
	InstancesCollector         *InstancesCollector
}

func Register(ctx context.Context, sub event.Subscriber, db storage.BrokerStorage, cfg Config, logger *slog.Logger) *RegisterContainer {
	logger = logger.With("from:", logPrefix)
	logger.Info("Registering metrics")
	opDurationCollector := NewOperationDurationCollector(logger)
	prometheus.MustRegister(opDurationCollector)

	opInstanceCollector := NewInstancesCollector(db.Instances(), logger)
	prometheus.MustRegister(opInstanceCollector)

	opResult := NewOperationsResults(db.Operations(), cfg, logger)
	opResult.StartCollector(ctx)

	opStats := NewOperationsStats(db.Operations(), cfg, logger)
	opStats.MustRegister()
	opStats.StartCollector(ctx)

	bindingStats := NewBindingStatsCollector(db.Bindings(), cfg.BindingsStatsPollingInterval, logger)
	bindingStats.MustRegister()
	bindingStats.StartCollector(ctx)

	bindDurationCollector := NewBindDurationCollector(logger)
	prometheus.MustRegister(bindDurationCollector)

	bindCrestedCollector := NewBindingCreationCollector()
	prometheus.MustRegister(bindCrestedCollector)

	runtimesInfoRequestsCollector := NewRuntimesInfoRequestsCollector()
	prometheus.MustRegister(runtimesInfoRequestsCollector)

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationFinished{}, opStats.Handler)
	sub.Subscribe(process.OperationFinished{}, opResult.Handler)

	sub.Subscribe(broker.BindRequestProcessed{}, bindDurationCollector.OnBindingExecuted)
	sub.Subscribe(broker.UnbindRequestProcessed{}, bindDurationCollector.OnUnbindingExecuted)
	sub.Subscribe(broker.BindingCreated{}, bindCrestedCollector.OnBindingCreated)

	sub.Subscribe(appinfo.RuntimesInfoRequest{}, runtimesInfoRequestsCollector.OnRequest)

	logger.Info(fmt.Sprintf("%s -> enabled", logPrefix))

	return &RegisterContainer{
		OperationResult:            opResult,
		OperationStats:             opStats,
		OperationDurationCollector: opDurationCollector,
		InstancesCollector:         opInstanceCollector,
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
	labels["error_category"] = string(op.LastError.GetComponent())
	labels["error_reason"] = string(op.LastError.GetReason())
	labels["error"] = op.LastError.Error()
	return labels
}
