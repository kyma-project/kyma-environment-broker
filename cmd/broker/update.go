package main

import (
	"context"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/update"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewUpdateProcessingQueue(ctx context.Context, manager *process.StagedManager, workersAmount int, db storage.BrokerStorage, inputFactory input.CreatorForPlan,
	publisher event.Publisher,
	cfg Config, k8sClientProvider K8sClientProvider, cli client.Client, logs *slog.Logger) *process.Queue {

	trialRegionsMapping, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	if err != nil {
		fatalOnError(err, logs)
	}

	manager.DefineStages([]string{"cluster", "btp-operator", "btp-operator-check", "check", "runtime_resource", "check_runtime_resource"})
	updateSteps := []struct {
		disabled  bool
		stage     string
		step      process.Step
		condition process.StepCondition
	}{
		{
			stage: "cluster",
			step:  update.NewInitialisationStep(db.Instances(), db.Operations()),
		},
		{
			stage:     "runtime_resource",
			step:      update.NewUpdateRuntimeStep(db.Operations(), cli, cfg.UpdateRuntimeResourceDelay, cfg.Provisioner, cfg.Broker.UseSmallerMachineTypes, trialRegionsMapping),
			condition: update.SkipForOwnClusterPlan,
		},
		{
			stage:     "check_runtime_resource",
			step:      steps.NewCheckRuntimeResourceStep(db.Operations(), cli, internal.RetryTuple{Timeout: cfg.Provisioner.RuntimeResourceStepTimeout, Interval: resourceStateRetryInterval}),
			condition: update.SkipForOwnClusterPlan,
		},
	}

	for _, step := range updateSteps {
		if !step.disabled {
			err := manager.AddStep(step.stage, step.step, step.condition)
			if err != nil {
				fatalOnError(err, logs)
			}
		}
	}
	queue := process.NewQueue(manager, logs, "update-processing", cfg.Broker.WorkerHealthCheckWarnInterval, cfg.Broker.WorkerHealthCheckInterval)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
