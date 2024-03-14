package metricsv2

import (
	"context"
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
)

// Exposer gathers metrics and keeps these in memory and exposes to prometheus for fetching, it gathers them by:
// listening in real time for events by "Handler"
// fetching data from database by "Job"

type Exposer interface {
	Handler(ctx context.Context, event interface{}) error
	Job(ctx context.Context)
}

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, instances storage.Instances, logger logrus.FieldLogger) {

	opDurationCollector := NewOperationDurationCollector()
	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(NewInstancesCollector(instances))

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)

	operationResult := NewOperationResult(ctx, operations, logger, time.Second*30, time.Hour*24*7)

	// test of metrics for upcoming new implementation
	operationsCounter := NewOperationsStats(operations, time.Second*30, logger)
	operationsCounter.MustRegister(ctx)

	sub.Subscribe(process.OperationCounting{}, operationsCounter.Handler)
	sub.Subscribe(process.DeprovisioningSucceeded{}, operationResult.Handler)
}
