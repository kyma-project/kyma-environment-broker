package main

import (
	"context"
	"time"

	orchestrationExt "github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterOrchestrationProcessingQueue(ctx context.Context, db storage.BrokerStorage, provisionerClient provisioner.Client,
	pub event.Publisher, inputFactory input.CreatorForPlan, icfg *upgrade_cluster.TimeSchedule, pollingInterval time.Duration,
	runtimeResolver orchestrationExt.RuntimeResolver, notificationBuilder notification.BundleBuilder, logs logrus.FieldLogger,
	cli client.Client, cfg Config, speedFactor int) *process.Queue {

	upgradeClusterManager := upgrade_cluster.NewManager(db.Operations(), pub, logs.WithField("upgradeCluster", "manager"))
	upgradeClusterInit := upgrade_cluster.NewInitialisationStep(db.Operations(), db.Orchestrations(), provisionerClient, inputFactory, icfg, notificationBuilder)
	upgradeClusterManager.InitStep(upgradeClusterInit)

	upgradeClusterSteps := []struct {
		disabled  bool
		weight    int
		step      upgrade_cluster.Step
		condition upgrade_cluster.StepCondition
	}{
		{
			weight:    1,
			step:      upgrade_cluster.NewLogSkippingUpgradeStep(db.Operations()),
			condition: provisioning.DoForOwnClusterPlanOnly,
		},
		{
			weight:    10,
			step:      upgrade_cluster.NewSendNotificationStep(db.Operations(), notificationBuilder),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			weight:    10,
			step:      upgrade_cluster.NewUpgradeClusterStep(db.Operations(), db.RuntimeStates(), provisionerClient, icfg),
			condition: provisioning.SkipForOwnClusterPlan,
		},
	}

	for _, step := range upgradeClusterSteps {
		if !step.disabled {
			upgradeClusterManager.AddStep(step.weight, step.step, step.condition)
		}
	}

	orchestrateClusterManager := manager.NewUpgradeClusterManager(db.Orchestrations(), db.Operations(), db.Instances(),
		upgradeClusterManager, runtimeResolver, pollingInterval, logs.WithField("upgradeCluster", "orchestration"),
		cli, cfg.OrchestrationConfig, notificationBuilder, speedFactor)
	queue := process.NewQueue(orchestrateClusterManager, logs)

	queue.Run(ctx.Done(), 3)

	return queue
}
