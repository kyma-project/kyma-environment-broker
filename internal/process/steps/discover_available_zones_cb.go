package steps

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
)

type DiscoverAvailableZonesCBStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	providerSpec     *configuration.ProviderSpec
	gardenerClient   *gardener.Client
	factory          hyperscalers.Factory
}

func NewDiscoverAvailableZonesCBStep(db storage.BrokerStorage, providerSpec *configuration.ProviderSpec, gardenerClient *gardener.Client, factory hyperscalers.Factory) *DiscoverAvailableZonesCBStep {
	step := &DiscoverAvailableZonesCBStep{
		instanceStorage: db.Instances(),
		providerSpec:    providerSpec,
		gardenerClient:  gardenerClient,
		factory:         factory,
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.KEBDependency)
	return step
}

func (s *DiscoverAvailableZonesCBStep) Name() string {
	return "Discover_Available_Zones_CredentialsBinding"
}

func (s *DiscoverAvailableZonesCBStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	provider := runtime.CloudProviderFromString(operation.ProviderValues.ProviderType)
	if !s.providerSpec.ZonesDiscovery(provider) {
		log.Info(fmt.Sprintf("Zones discovery disabled for provider %s, skipping", provider))
		return operation, 0, nil
	}
	if len(operation.DiscoveredZones) > 0 {
		log.Info("Available zones already discovered, skipping")
		return operation, 0, nil
	}

	instance, err := s.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		if dberr.IsNotFound(err) {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("instance %s does not exists", operation.InstanceID), err, log)
		}
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get instance %s", operation.InstanceID), err, 10*time.Second, time.Minute, log)
	}

	subscriptionSecretName := instance.SubscriptionSecretName
	if subscriptionSecretName == "" {
		if operation.ProvisioningParameters.Parameters.TargetSecret == nil {
			return s.operationManager.OperationFailed(operation, "subscription secret name is missing", nil, log)
		}
		subscriptionSecretName = *operation.ProvisioningParameters.Parameters.TargetSecret
	}

	credentialsBinding, err := s.gardenerClient.GetCredentialsBinding(subscriptionSecretName)
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get credentials binding %s", subscriptionSecretName), err, 10*time.Second, time.Minute, log)
	}

	secret, err := s.gardenerClient.GetSecret(credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName())
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get secret %s/%s", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName()), err, 10*time.Second, time.Minute, log)
	}

	log.Info(fmt.Sprintf("discovering zones using credentials binding %s region=%s", subscriptionSecretName, operation.ProviderValues.Region))

	// Always use a per-call client with the exact Kyma-specific secret to ensure
	// zone discovery reflects the actual subscription restrictions for this instance.
	client, err := s.factory.NewPerCallFromSecret(context.Background(), provider, secret, operation.ProviderValues.Region)
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to create %s client", provider), err, 10*time.Second, time.Minute, log)
	}

	discoveredZones := make(map[string][]string)
	switch operation.Type {
	case internal.OperationTypeProvision:
		discoveredZones[DefaultIfParamNotSet(operation.ProviderValues.DefaultMachineType, operation.ProvisioningParameters.Parameters.MachineType)] = []string{}
		for _, pool := range operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools {
			discoveredZones[pool.MachineType] = []string{}
		}
	case internal.OperationTypeUpdate:
		for _, pool := range operation.UpdatingParameters.AdditionalWorkerNodePools {
			discoveredZones[pool.MachineType] = []string{}
		}
	}

	for machineType := range discoveredZones {
		zones, err := client.AvailableZones(context.Background(), machineType)
		if err != nil {
			return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get available zones for machine type %s", machineType), err, 10*time.Second, time.Minute, log)
		}
		rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
		log.Info(fmt.Sprintf("Available zones for machine type %s in region %s: %v", machineType, operation.ProviderValues.Region, zones))
		discoveredZones[machineType] = zones
	}

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.DiscoveredZones = discoveredZones
	}, log)
}

func DefaultIfParamNotSet[T interface{}](d T, param *T) T {
	if param == nil {
		return d
	}
	return *param
}
