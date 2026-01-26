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

	useCredentialsBinding := strings.ToLower(cfg.SubscriptionGardenerResource) == credentialsBinding

	provisioningSteps := []struct {
		disabled  bool
		step      process.Step
		condition process.StepCondition
	}{
		{
			step: provisioning.NewStartStep(db.Operations(), db.Instances()),
		},
		{
			step: steps.NewInitKymaTemplate(db.Operations(), config.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, config.RuntimeConfigurationRequiredFields)),
		},
		{
			step: provisioning.NewOverrideKymaModules(db.Operations()),
		},
		{
			step: steps.NewHolderStep(cfg.HoldHapSteps,
				provisioning.NewResolveSubscriptionSecretStep(db, gardenerClient, rulesService, internal.RetryTuple{Timeout: resolveSubscriptionSecretTimeout, Interval: resolveSubscriptionSecretRetryInterval})),
			disabled: useCredentialsBinding,
		},
		{
			step: steps.NewHolderStep(cfg.HoldHapSteps,
				provisioning.NewResolveCredentialsBindingStep(db, gardenerClient, rulesService, internal.RetryTuple{Timeout: resolveSubscriptionSecretTimeout, Interval: resolveSubscriptionSecretRetryInterval})),
			disabled: !useCredentialsBinding,
		},
		{
			step:     steps.NewDiscoverAvailableZonesStep(db, providerSpec, gardenerClient, awsClientFactory),
			disabled: useCredentialsBinding,
		},
		{
			step:     steps.NewDiscoverAvailableZonesCBStep(db, providerSpec, gardenerClient, awsClientFactory),
			disabled: !useCredentialsBinding,
		},
		{
			step: provisioning.NewGenerateRuntimeIDStep(db.Operations(), db.Instances()),
		},
		{
			step: provisioning.NewCreateResourceNamesStep(db.Operations()),
		},
		{
			step: provisioning.NewCreateRuntimeResourceStep(db, k8sClient, cfg.InfrastructureManager, defaultOIDC, workersProvider, providerSpec),
		},
		{
			step: steps.NewCheckRuntimeResourceProvisioningStep(db.Operations(), k8sClient, internal.RetryTuple{Timeout: cfg.StepTimeouts.CheckRuntimeResourceCreate, Interval: resourceStateRetryInterval}, provisioningTakesLongThreshold),
		},
		{
			condition: provisioning.WhenBTPOperatorCredentialsProvided,
			step:      provisioning.NewInjectBTPOperatorCredentialsStep(db.Operations(), k8sClientProvider),
		},
		{
			step: provisioning.NewApplyKymaStep(db.Operations(), k8sClient),
		},
	}
	var stages []string
	for _, step := range provisioningSteps {
		if !step.disabled {
			stages = append(stages, step.step.Name())
		}
	}
	provisionManager.DefineStages(stages)
	for _, step := range provisioningSteps {
		if !step.disabled {
			err := provisionManager.AddStep(step.step.Name(), step.step, step.condition)
			if err != nil {
				fatalOnError(err, logs)
			}
		}
	}

	queue := process.NewQueue(provisionManager, logs, "provisioning")
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
