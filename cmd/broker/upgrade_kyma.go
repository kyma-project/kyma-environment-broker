package main

import (
	"context"
	"time"

	orchestrationExt "github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKymaOrchestrationProcessingQueue(ctx context.Context, db storage.BrokerStorage,
	provisionerClient provisioner.Client, pub event.Publisher,
	inputFactory input.CreatorForPlan, icfg *upgrade_kyma.TimeSchedule, pollingInterval time.Duration,
	runtimeResolver orchestrationExt.RuntimeResolver,
	upgradeEvalManager *avs.EvaluationManager, cfg *Config, internalEvalAssistant *avs.InternalEvalAssistant,
	notificationBuilder notification.BundleBuilder, logs logrus.FieldLogger,
	cli client.Client, speedFactor int) *process.Queue {

	upgradeKymaManager := upgrade_kyma.NewManager(db.Operations(), pub, logs.WithField("upgradeKyma", "manager"))
	upgradeKymaInit := upgrade_kyma.NewInitialisationStep(db.Operations(), db.Orchestrations(), db.Instances(),
		provisionerClient, inputFactory, upgradeEvalManager, icfg, notificationBuilder)

	upgradeKymaManager.InitStep(upgradeKymaInit)
	upgradeKymaSteps := []struct {
		disabled bool
		weight   int
		step     upgrade_kyma.Step
		cnd      upgrade_kyma.StepCondition
	}{
		{
			weight: 1,
			step:   steps.InitKymaTemplateUpgradeKyma(db.Operations()),
		},
		{
			weight:   2,
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			step:     upgrade_kyma.NewApplyKymaStep(db.Operations(), cli),
		},
		{
			weight: 8,
			step:   upgrade_kyma.NewSendNotificationStep(db.Operations(), notificationBuilder),
		},
	}
	for _, step := range upgradeKymaSteps {
		if !step.disabled {
			upgradeKymaManager.AddStep(step.weight, step.step, step.cnd)
		}
	}

	orchestrateKymaManager := manager.NewUpgradeKymaManager(db.Orchestrations(), db.Operations(), db.Instances(),
		upgradeKymaManager, runtimeResolver, pollingInterval, logs.WithField("upgradeKyma", "orchestration"),
		cli, &cfg.OrchestrationConfig, notificationBuilder, speedFactor)
	queue := process.NewQueue(orchestrateKymaManager, logs)

	queue.Run(ctx.Done(), 3)

	return queue
}
