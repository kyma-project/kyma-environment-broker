package main

import (
	"context"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDeprovisioningProcessingQueue(ctx context.Context, workersAmount int, deprovisionManager *process.StagedManager,
	cfg *Config, db storage.BrokerStorage, pub event.Publisher,
	provisionerClient provisioner.Client,
	edpClient deprovisioning.EDPClient, accountProvider hyperscaler.AccountProvider,
	k8sClientProvider K8sClientProvider, cli client.Client, configProvider input.ConfigurationProvider, logs logrus.FieldLogger) *process.Queue {

	deprovisioningSteps := []struct {
		disabled bool
		step     process.Step
	}{
		{
			step: deprovisioning.NewInitStep(db.Operations(), db.Instances(), 12*time.Hour),
		},
		{
			step: deprovisioning.NewBTPOperatorCleanupStep(db.Operations(), k8sClientProvider),
		},
		{
			step:     deprovisioning.NewEDPDeregistrationStep(db.Operations(), db.Instances(), edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			step:     deprovisioning.NewDeleteKymaResourceStep(db.Operations(), db.Instances(), cli, configProvider),
		},
		{
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			step:     deprovisioning.NewCheckKymaResourceDeletedStep(db.Operations(), cli, cfg.KymaResourceDeletionTimeout),
		},
		{
			step: deprovisioning.NewDeleteRuntimeResourceStep(db.Operations(), cli),
		},
		{
			step: deprovisioning.NewCheckRuntimeResourceDeletionStep(db.Operations(), cli),
		},
		{
			step: deprovisioning.NewDeleteGardenerClusterStep(db.Operations(), cli, db.Instances()),
		},
		{
			step: deprovisioning.NewCheckGardenerClusterDeletedStep(db.Operations(), cli),
		},
		{
			step: deprovisioning.NewRemoveRuntimeStep(db.Operations(), db.Instances(), provisionerClient, cfg.Provisioner.DeprovisioningTimeout),
		},
		{
			step: deprovisioning.NewCheckRuntimeRemovalStep(db.Operations(), db.Instances(), provisionerClient, cfg.Provisioner.DeprovisioningTimeout),
		},
		{
			step: deprovisioning.NewReleaseSubscriptionStep(db.Operations(), db.Instances(), accountProvider),
		},
		{
			disabled: true,
			step:     steps.DeleteKubeconfig(db.Operations(), cli),
		},
		{
			disabled: !cfg.ArchiveEnabled,
			step:     deprovisioning.NewArchivingStep(db.Operations(), db.Instances(), db.InstancesArchived(), cfg.ArchiveDryRun),
		},
		{
			step: deprovisioning.NewRemoveInstanceStep(db.Instances(), db.Operations()),
		},
		{
			disabled: !cfg.CleaningEnabled,
			step:     deprovisioning.NewCleanStep(db.Operations(), db.RuntimeStates(), cfg.CleaningDryRun),
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

	queue := process.NewQueue(deprovisionManager, logs)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
