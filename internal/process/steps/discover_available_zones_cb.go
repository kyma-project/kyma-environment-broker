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
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
)

type DiscoverAvailableZonesCBStep struct {
	operationManager *process.OperationManager
	operationStorage storage.Operations
	instanceStorage  storage.Instances
	providerSpec     *configuration.ProviderSpec
	gardenerClient   *gardener.Client
	awsClientFactory aws.ClientFactory
}

func NewDiscoverAvailableZonesCBStep(db storage.BrokerStorage, providerSpec *configuration.ProviderSpec, gardenerClient *gardener.Client, awsClientFactory aws.ClientFactory) *DiscoverAvailableZonesCBStep {
	step := &DiscoverAvailableZonesCBStep{
		operationStorage: db.Operations(),
		instanceStorage:  db.Instances(),
		providerSpec:     providerSpec,
		gardenerClient:   gardenerClient,
		awsClientFactory: awsClientFactory,
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.KEBDependency)
	return step
}

func (s *DiscoverAvailableZonesCBStep) Name() string {
	return "Discover_Available_Zones_CredentialsBinding"
}

func (s *DiscoverAvailableZonesCBStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if !s.providerSpec.ZonesDiscovery(runtime.CloudProviderFromString(operation.ProviderValues.ProviderType)) {
		log.Info(fmt.Sprintf("Zones discovery disabled for provider %s, skipping", runtime.CloudProviderFromString(operation.ProviderValues.ProviderType)))
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
	accessKeyID, secretAccessKey, err := aws.ExtractCredentials(secret)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "failed to extract AWS credentials", err, log)
	}

	client, err := s.awsClientFactory.New(context.Background(), accessKeyID, secretAccessKey, operation.ProviderValues.Region)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "unable to create AWS client", err, 10*time.Second, time.Minute, log)
	}

	operation.DiscoveredZones = make(map[string][]string)
	if operation.Type == internal.OperationTypeProvision {
		operation.DiscoveredZones[DefaultIfParamNotSet(operation.ProviderValues.DefaultMachineType, operation.ProvisioningParameters.Parameters.MachineType)] = []string{}
		for _, pool := range operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools {
			operation.DiscoveredZones[pool.MachineType] = []string{}
		}
	} else if operation.Type == internal.OperationTypeUpdate {
		for _, pool := range operation.UpdatingParameters.AdditionalWorkerNodePools {
			operation.DiscoveredZones[pool.MachineType] = []string{}
		}
	}

	for machineType := range operation.DiscoveredZones {
		zones, err := client.AvailableZones(context.Background(), machineType)
		if err != nil {
			return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get available zones for machine type %s", machineType), err, 10*time.Second, time.Minute, log)
		}
		rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
		log.Info(fmt.Sprintf("Available zones for machine type %s: %v", machineType, zones))
		operation.DiscoveredZones[machineType] = zones
	}

	return operation, 0, nil
}

func DefaultIfParamNotSet[T interface{}](d T, param *T) T {
	if param == nil {
		return d
	}
	return *param
}
