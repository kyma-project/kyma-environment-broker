package metricsv2

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
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
	OperationResultPoolingInterval                  time.Duration `envconfig:"default=1m"`
	OperationStatsPoolingInterval                   time.Duration `envconfig:"default=1m"`
	OperationResultFinishedOperationRetentionPeriod time.Duration `envconfig:"default=3h"`
}

type RegisterContainer struct {
	OperationResult            *operationsResult
	OperationStats             *OperationStats
	OperationDurationCollector *OperationDurationCollector
	InstancesCollector         *InstancesCollector
}

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, instances storage.Instances, cfg Config, logger logrus.FieldLogger) *RegisterContainer {
	logger = logger.WithField("from:", logPrefix)
	logrus.Infof("Registering metricsv2")
	opDurationCollector := NewOperationDurationCollector(logger)
	prometheus.MustRegister(opDurationCollector)

	opInstanceCollector := NewInstancesCollector(instances, logger)
	prometheus.MustRegister(opInstanceCollector)

	opResult := NewOperationResult(ctx, operations, cfg, logger)

	opStats := NewOperationsStats(operations, cfg, logger)
	opStats.MustRegister(ctx)

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationFinished{}, opStats.Handler)
	sub.Subscribe(process.OperationFinished{}, opResult.Handler)

	logger.Infof(fmt.Sprintf("%s -> enabled", logPrefix))

	return &RegisterContainer{
		OperationResult:            opResult,
		OperationStats:             opStats,
		OperationDurationCollector: opDurationCollector,
		InstancesCollector:         opInstanceCollector,
	}
}
