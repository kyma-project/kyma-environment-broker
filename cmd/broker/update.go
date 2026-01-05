package main

import (
	"context"
	"log/slog"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/process/update"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewUpdateProcessingQueue(ctx context.Context, manager *process.StagedManager, workersAmount int, db storage.BrokerStorage,
	cfg Config, kcpClient client.Client, logs *slog.Logger, workersProvider *workers.Provider, schemaService *broker.SchemaService, planSpec *configuration.PlanSpecifications, configProvider config.Provider,
	providerSpec *configuration.ProviderSpec, gardenerClient *gardener.Client, awsClientFactory aws.ClientFactory) *process.Queue {

	trialRegionsMapping, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	if err != nil {
		fatalOnError(err, logs)
	}

	regions, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	valuesProvider := provider.NewPlanSpecificValuesProvider(cfg.InfrastructureManager, regions, schemaService, planSpec)

	useCredentialsBinding := strings.ToLower(cfg.SubscriptionGardenerResource) == "credentialsbinding"

	manager.DefineStages([]string{"cluster", "btp-operator", "btp-operator-check", "check", "runtime_resource", "check_runtime_resource", "kyma_resource"})
	updateSteps := []struct {
		disabled  bool
		stage     string
		step      process.Step
		condition process.StepCondition
	}{
		{
			stage: "cluster",
			step:  update.NewInitialisationStep(db),
		},
		{
			stage:     "runtime_resource",
			step:      steps.NewDiscoverAvailableZonesStep(db, providerSpec, gardenerClient, awsClientFactory),
			condition: update.SkipForOwnClusterPlan,
			disabled:  useCredentialsBinding,
		},
		{
			stage:     "runtime_resource",
			step:      steps.NewDiscoverAvailableZonesCBStep(db, providerSpec, gardenerClient, awsClientFactory),
			condition: update.SkipForOwnClusterPlan,
			disabled:  !useCredentialsBinding,
		},
		{
			stage:     "runtime_resource",
			step:      update.NewUpdateRuntimeStep(db, kcpClient, cfg.UpdateRuntimeResourceDelay, cfg.InfrastructureManager, trialRegionsMapping, workersProvider, valuesProvider),
			condition: update.SkipForOwnClusterPlan,
		},
		{
			stage:     "check_runtime_resource",
			step:      steps.NewCheckRuntimeResourceStep(db.Operations(), kcpClient, internal.RetryTuple{Timeout: cfg.StepTimeouts.CheckRuntimeResourceUpdate, Interval: resourceStateRetryInterval}),
			condition: update.SkipForOwnClusterPlan,
		},
		{
			stage: "kyma_resource",
			step:  update.NewUpdateKymaStep(db, kcpClient, config.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, config.RuntimeConfigurationRequiredFields)),
		},
	}

	for _, step := range updateSteps {
		if !step.disabled {
			err = manager.AddStep(step.stage, step.step, step.condition)
			if err != nil {
				fatalOnError(err, logs)
			}
		}
	}
	queue := process.NewQueue(manager, logs, "update-processing")
	queue.Run(ctx.Done(), workersAmount)

	return queue
}
