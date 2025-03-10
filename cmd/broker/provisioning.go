package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const resourceStateRetryInterval = 10 * time.Second

func NewProvisioningProcessingQueue(ctx context.Context, provisionManager *process.StagedManager, workersAmount int, cfg *Config,
	db storage.BrokerStorage, configProvider input.ConfigurationProvider,
	edpClient provisioning.EDPClient, accountProvider hyperscaler.AccountProvider,
	k8sClientProvider provisioning.K8sClientProvider, cli client.Client, defaultOIDC pkg.OIDCConfigDTO, logs *slog.Logger, rulesService *rules.RulesService) *process.Queue {

	trialRegionsMapping, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	if err != nil {
		fatalOnError(err, logs)
	}

	provisionManager.DefineStages([]string{startStageName, createRuntimeStageName,
		checkKymaStageName, createKymaResourceStageName})
	/*
			The provisioning process contains the following stages:
			1. "start" - changes the state from pending to in progress if no deprovisioning is ongoing.
			2. "create_runtime" - collects all information needed to make an input for the Provisioner request as overrides and labels.
			Those data is collected using an InputCreator which is not persisted. That's why all steps which prepares such data must be in the same stage as "create runtime step".
		    All steps which requires InputCreator must be run in this stage.
			3. "check_kyma" - checks if the Kyma is installed
			4. "post_actions" - all steps which must be executed after the runtime is provisioned

			Once the stage is done it will never be retried.
	*/

	fatalOnError(err, logs)

	provisioningSteps := []struct {
		disabled  bool
		stage     string
		step      process.Step
		condition process.StepCondition
	}{
		{
			stage: startStageName,
			step:  provisioning.NewStartStep(db.Operations(), db.Instances()),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewInitProviderValuesStep(db.Operations(), cfg.Provisioner, trialRegionsMapping, cfg.Broker.UseSmallerMachineTypes),
		},
		{
			stage: createRuntimeStageName,
			step:  steps.NewInitKymaTemplate(db.Operations(), configProvider),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewOverrideKymaModules(db.Operations()),
		},
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewResolveCredentialsStep(db.Operations(), accountProvider, rulesService),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewEDPRegistrationStep(db.Operations(), edpClient, cfg.EDP),
			disabled:  cfg.EDP.Disabled,
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewGenerateRuntimeIDStep(db.Operations(), db.Instances()),
		},
		// postcondition: operation.RuntimeID is set
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewCreateResourceNamesStep(db.Operations()),
		},
		// postcondition: operation.KymaResourceName, operation.RuntimeResourceName is set
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateRuntimeResourceStep(db.Operations(), db.Instances(), cli, cfg.Provisioner, defaultOIDC),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage:     createRuntimeStageName,
			step:      steps.NewCheckRuntimeResourceStep(db.Operations(), cli, internal.RetryTuple{Timeout: cfg.Provisioner.RuntimeResourceStepTimeout, Interval: resourceStateRetryInterval}),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{ // TODO: this step must be removed when kubeconfig is created by IM and own_cluster plan is permanently removed
			disabled:  cfg.LifecycleManagerIntegrationDisabled,
			stage:     createRuntimeStageName,
			step:      steps.SyncKubeconfig(db.Operations(), cli),
			condition: provisioning.DoForOwnClusterPlanOnly,
		},
		{ // must be run after the secret with kubeconfig is created ("syncKubeconfig")
			condition: provisioning.WhenBTPOperatorCredentialsProvided,
			stage:     createRuntimeStageName,
			step:      provisioning.NewInjectBTPOperatorCredentialsStep(db.Operations(), k8sClientProvider),
		},
		{
			disabled: cfg.LifecycleManagerIntegrationDisabled,
			stage:    createKymaResourceStageName,
			step:     provisioning.NewApplyKymaStep(db.Operations(), cli),
		},
	}
	for _, step := range provisioningSteps {
		if !step.disabled {
			err := provisionManager.AddStep(step.stage, step.step, step.condition)
			if err != nil {
				fatalOnError(err, logs)
			}
		}
	}

	queue := process.NewQueue(provisionManager, logs, "provisioning", cfg.Broker.WorkerHealthCheckWarnInterval, cfg.Broker.WorkerHealthCheckInterval)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
