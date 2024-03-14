package metrics

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, instanceStatsGetter InstancesStatsGetter, logger logrus.FieldLogger) {
	opResultCollector := NewOperationResultCollector()
	opDurationCollector := NewOperationDurationCollector()
	stepResultCollector := NewStepResultCollector()
	prometheus.MustRegister(opResultCollector, opDurationCollector, stepResultCollector)
	prometheus.MustRegister(NewOperationsCollector(operations))
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	sub.Subscribe(process.ProvisioningStepProcessed{}, opResultCollector.OnProvisioningStepProcessed)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opResultCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.UpgradeKymaStepProcessed{}, opResultCollector.OnUpgradeKymaStepProcessed)
	sub.Subscribe(process.UpgradeClusterStepProcessed{}, opResultCollector.OnUpgradeClusterStepProcessed)
	sub.Subscribe(process.ProvisioningSucceeded{}, opResultCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.ProvisioningStepProcessed{}, stepResultCollector.OnProvisioningStepProcessed)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, stepResultCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationStepProcessed{}, stepResultCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationStepProcessed{}, opResultCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opResultCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)

	StartOpsMetricService(ctx, operations, logger)
}
