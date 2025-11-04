package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	resourceStateRetryInterval             = 10 * time.Second
	resolveSubscriptionSecretRetryInterval = 10 * time.Second

	resolveSubscriptionSecretTimeout = 1 * time.Minute
)

func NewProvisioningProcessingQueue(ctx context.Context, provisionManager *process.StagedManager, workersAmount int, cfg *Config,
	db storage.BrokerStorage, configProvider config.Provider,
	k8sClientProvider provisioning.K8sClientProvider, k8sClient client.Client, gardenerClient *gardener.Client, defaultOIDC pkg.OIDCConfigDTO, logs *slog.Logger, rulesService *rules.RulesService,
	workersProvider *workers.Provider, providerSpec *configuration.ProviderSpec, awsClientFactory aws.ClientFactory) *process.Queue {

	useCredentialsBinding := strings.ToLower(cfg.SubscriptionGardenerResource) == "credentialsbinding"

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
			step:  steps.NewInitKymaTemplate(db.Operations(), config.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, config.RuntimeConfigurationRequiredFields)),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewOverrideKymaModules(db.Operations()),
		},
		{
			stage: createRuntimeStageName,
			step: steps.NewHolderStep(cfg.HoldHapSteps,
				provisioning.NewResolveSubscriptionSecretStep(db, gardenerClient, rulesService, internal.RetryTuple{Timeout: resolveSubscriptionSecretTimeout, Interval: resolveSubscriptionSecretRetryInterval})),
			condition: provisioning.SkipForOwnClusterPlan,
			disabled:  useCredentialsBinding,
		},
		{
			stage: createRuntimeStageName,
			step: steps.NewHolderStep(cfg.HoldHapSteps,
				provisioning.NewResolveCredentialsBindingStep(db, gardenerClient, rulesService, internal.RetryTuple{Timeout: resolveSubscriptionSecretTimeout, Interval: resolveSubscriptionSecretRetryInterval})),
			condition: provisioning.SkipForOwnClusterPlan,
			disabled:  !useCredentialsBinding,
		},
		{
			stage:     createRuntimeStageName,
			step:      steps.NewDiscoverAvailableZonesStep(db, providerSpec, gardenerClient, awsClientFactory),
			condition: provisioning.SkipForOwnClusterPlan,
			disabled:  useCredentialsBinding,
		},
		{
			stage:    createRuntimeStageName,
			step:     steps.NewDiscoverAvailableZonesCBStep(db, providerSpec, gardenerClient, awsClientFactory),
			disabled: !useCredentialsBinding,
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewGenerateRuntimeIDStep(db.Operations(), db.Instances()),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewCreateResourceNamesStep(db.Operations()),
		},
		{
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateRuntimeResourceStep(db, k8sClient, cfg.InfrastructureManager, defaultOIDC, workersProvider, providerSpec),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{
			stage:     createRuntimeStageName,
			step:      steps.NewCheckRuntimeResourceProvisioningStep(db.Operations(), k8sClient, internal.RetryTuple{Timeout: cfg.StepTimeouts.CheckRuntimeResourceCreate, Interval: resourceStateRetryInterval}, provisioningTakesLongThreshold),
			condition: provisioning.SkipForOwnClusterPlan,
		},
		{ // TODO: this step must be removed when kubeconfig is created by IM and own_cluster plan is permanently removed
			stage:     createRuntimeStageName,
			step:      steps.SyncKubeconfig(db.Operations(), k8sClient),
			condition: provisioning.DoForOwnClusterPlanOnly,
		},
		{ // must be run after the secret with kubeconfig is created ("syncKubeconfig")
			condition: provisioning.WhenBTPOperatorCredentialsProvided,
			stage:     createRuntimeStageName,
			step:      provisioning.NewInjectBTPOperatorCredentialsStep(db.Operations(), k8sClientProvider),
		},
		{
			stage: createKymaResourceStageName,
			step:  provisioning.NewApplyKymaStep(db.Operations(), k8sClient),
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

	queue := process.NewQueue(provisionManager, logs, "provisioning")
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
