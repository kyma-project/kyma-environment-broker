package main

import (
	"context"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/update"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewUpdateProcessingQueue(ctx context.Context, manager *process.StagedManager, workersAmount int, db storage.BrokerStorage, inputFactory input.CreatorForPlan,
	provisionerClient provisioner.Client, publisher event.Publisher, runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator, runtimeStatesDb storage.RuntimeStates,
	runtimeProvider input.ComponentListProvider, reconcilerClient reconciler.Client, cfg Config, k8sClientProvider K8sClientProvider, cli client.Client, logs logrus.FieldLogger) *process.Queue {

	requiresReconcilerUpdate := update.RequiresReconcilerUpdate
	if cfg.ReconcilerIntegrationDisabled {
		requiresReconcilerUpdate = func(op internal.Operation) bool { return false }
	}
	manager.DefineStages([]string{"cluster", "btp-operator", "btp-operator-check", "check"})
	updateSteps := []struct {
		stage     string
		step      process.Step
		condition process.StepCondition
	}{
		{
			stage: "cluster",
			step:  update.NewInitialisationStep(db.Instances(), db.Operations(), runtimeVerConfigurator, inputFactory),
		},
		{
			stage:     "cluster",
			step:      update.NewUpgradeShootStep(db.Operations(), db.RuntimeStates(), provisionerClient),
			condition: update.SkipForOwnClusterPlan,
		},
		{
			stage: "btp-operator",
			step:  update.NewInitKymaVersionStep(db.Operations(), runtimeVerConfigurator, runtimeStatesDb),
		},
		{
			stage:     "btp-operator",
			step:      update.NewBTPOperatorOverridesStep(db.Operations(), runtimeProvider),
			condition: update.RequiresBTPOperatorCredentials,
		},
		{
			stage:     "btp-operator",
			step:      update.NewApplyReconcilerConfigurationStep(db.Operations(), db.RuntimeStates(), reconcilerClient),
			condition: requiresReconcilerUpdate,
		},
		{
			stage:     "btp-operator-check",
			step:      update.NewCheckReconcilerState(db.Operations(), reconcilerClient),
			condition: update.CheckReconcilerStatus,
		},
		{
			stage:     "check",
			step:      update.NewCheckStep(db.Operations(), provisionerClient, 40*time.Minute),
			condition: update.SkipForOwnClusterPlan,
		},
	}

	for _, step := range updateSteps {
		err := manager.AddStep(step.stage, step.step, step.condition)
		if err != nil {
			fatalOnError(err, logs)
		}
	}
	queue := process.NewQueue(manager, logs)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
