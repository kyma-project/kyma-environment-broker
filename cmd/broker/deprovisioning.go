package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/config"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDeprovisioningProcessingQueue(ctx context.Context, workersAmount int, deprovisionManager *process.StagedManager,
	cfg *Config, db storage.BrokerStorage,
	edpClient deprovisioning.EDPClient, accountProvider hyperscaler.AccountProvider,
	k8sClientProvider K8sClientProvider, kcpClient client.Client, configProvider config.ConfigurationProvider, logs *slog.Logger) *process.Queue {

	deprovisioningSteps := []struct {
		disabled bool
		step     process.Step
	}{
		{
			step: deprovisioning.NewInitStep(db, 12*time.Hour),
		},
		{
			step: deprovisioning.NewBTPOperatorCleanupStep(db, k8sClientProvider),
		},
		{
			step:     deprovisioning.NewEDPDeregistrationStep(db, edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			step:     deprovisioning.NewDeleteKymaResourceStep(db, kcpClient, configProvider),
		},
		{
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			step:     deprovisioning.NewCheckKymaResourceDeletedStep(db, kcpClient, cfg.KymaResourceDeletionTimeout),
		},
		{
			step: deprovisioning.NewDeleteRuntimeResourceStep(db, kcpClient),
		},
		{
			step: deprovisioning.NewCheckRuntimeResourceDeletionStep(db, kcpClient, cfg.StepTimeouts.CheckRuntimeResourceDeletion),
		},
		{
			step: deprovisioning.NewReleaseSubscriptionStep(db, accountProvider),
		},
		{
			disabled: !cfg.ArchiveEnabled,
			step:     deprovisioning.NewArchivingStep(db, cfg.ArchiveDryRun),
		},
		{
			step: deprovisioning.NewRemoveInstanceStep(db),
		},
		{
			disabled: !cfg.CleaningEnabled,
			step:     deprovisioning.NewCleanStep(db, cfg.CleaningDryRun),
		},
	}
	var stages []string
	for _, step := range deprovisioningSteps {
		if !step.disabled {
			stages = append(stages, step.step.Name())
		}
	}
	deprovisionManager.DefineStages(stages)
	for _, step := range deprovisioningSteps {
		if !step.disabled {
			err := deprovisionManager.AddStep(step.step.Name(), step.step, nil)
			fatalOnError(err, logs)
		}
	}

	queue := process.NewQueue(deprovisionManager, logs, "deprovisioning", cfg.Broker.WorkerHealthCheckWarnInterval, cfg.Broker.WorkerHealthCheckInterval)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
