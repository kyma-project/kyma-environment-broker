package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"

	"k8s.io/client-go/dynamic"

	"github.com/kyma-project/kyma-environment-broker/internal/config"

	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDeprovisioningProcessingQueue(ctx context.Context, workersAmount int, deprovisionManager *process.StagedManager,
	cfg *Config, db storage.BrokerStorage,
	k8sClientProvider K8sClientProvider, kcpClient client.Client, configProvider config.Provider, gardenerClient dynamic.Interface, gardenerNamespace string, logs *slog.Logger) *process.Queue {

	useCredentialsBinding := strings.ToLower(cfg.SubscriptionGardenerResource) == credentialsBinding

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
			step: deprovisioning.NewDeleteKymaResourceStep(db, kcpClient, config.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, config.RuntimeConfigurationRequiredFields)),
		},
		{
			step: deprovisioning.NewCheckKymaResourceDeletedStep(db, kcpClient),
		},
		{
			step: deprovisioning.NewDeleteRuntimeResourceStep(db, kcpClient),
		},
		{
			step: deprovisioning.NewCheckRuntimeResourceDeletionStep(db, kcpClient, cfg.StepTimeouts.CheckRuntimeResourceDeletion),
		},
		{
			disabled: useCredentialsBinding,
			step: steps.NewHolderStep(cfg.HoldHapSteps,
				deprovisioning.NewFreeSubscriptionStep(db.Operations(), db.Instances(), gardenerClient, gardenerNamespace)),
		},
		{
			disabled: !useCredentialsBinding,
			step: steps.NewHolderStep(cfg.HoldHapSteps,
				deprovisioning.NewFreeCredentialsBindingStep(db.Operations(), db.Instances(), gardenerClient, gardenerNamespace)),
		},
		{
			step: deprovisioning.NewArchivingStep(db),
		},
		{
			step: deprovisioning.NewRemoveInstanceStep(db),
		},
		{
			step: deprovisioning.NewCleanStep(db),
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

	queue := process.NewQueue(deprovisionManager, logs, "deprovisioning")
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
