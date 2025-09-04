package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type DiscoverAvailableZonesStep struct {
	operationManager *process.OperationManager
	opStorage        storage.Operations
	providerSpec     *configuration.ProviderSpec
	gardenerClient   *gardener.Client
	awsClientFactory aws.ClientFactory
}

func NewDiscoverAvailableZonesStep(os storage.Operations, providerSpec *configuration.ProviderSpec, gardenerClient *gardener.Client, awsClientFactory aws.ClientFactory) *DiscoverAvailableZonesStep {
	step := &DiscoverAvailableZonesStep{
		opStorage:        os,
		providerSpec:     providerSpec,
		gardenerClient:   gardenerClient,
		awsClientFactory: awsClientFactory,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *DiscoverAvailableZonesStep) Name() string {
	return "Discover_Available_Zones"
}

func (s *DiscoverAvailableZonesStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	//if !s.providerSpec.ZonesDiscovery(operation.ProvisioningParameters.PlatformProvider) {
	//	log.Info(fmt.Sprintf("Zones discovery disabled for provider %s, skipping", operation.ProvisioningParameters.PlatformProvider))
	//	return operation, 0, nil
	//}

	if operation.ProvisioningParameters.Parameters.TargetSecret == nil {
		return s.operationManager.OperationFailed(operation, "target secret is missing", nil, log)
	}
	if operation.ProvisioningParameters.Parameters.Region == nil {
		return s.operationManager.OperationFailed(operation, "region is missing", nil, log)
	}

	secret, err := s.gardenerClient.GetSecret(*operation.ProvisioningParameters.Parameters.TargetSecret)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "unable to get secret", err, 10*time.Second, time.Minute, log)
	}
	accessKeyID, secretAccessKey, err := aws.ExtractCredentials(secret)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "failed to extract AWS credentials", err, log)
	}

	client, err := s.awsClientFactory.New(context.Background(), accessKeyID, secretAccessKey, *operation.ProvisioningParameters.Parameters.Region)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "unable to create AWS client", err, 10*time.Second, time.Minute, log)
	}

	operation.DiscoveredZones = make(map[string][]string)
	if operation.Type == internal.OperationTypeProvision {
		if operation.ProvisioningParameters.Parameters.MachineType != nil {
			operation.DiscoveredZones[*operation.ProvisioningParameters.Parameters.MachineType] = []string{}
		}
		for _, pool := range operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools {
			operation.DiscoveredZones[pool.MachineType] = []string{}
		}
	} else if operation.Type == internal.OperationTypeUpdate {
		for _, pool := range operation.UpdatingParameters.AdditionalWorkerNodePools {
			operation.DiscoveredZones[pool.MachineType] = []string{}
		}
	}

	for machineType, _ := range operation.DiscoveredZones {
		zones, err := client.AvailableZones(context.Background(), machineType)
		if err != nil {
			return s.operationManager.RetryOperation(operation, fmt.Sprintf("unable to get available zones for machine type %s", machineType), err, 10*time.Second, time.Minute, log)
		}
		log.Info(fmt.Sprintf("Available zones for machine type %s: %v", machineType, zones))
		operation.DiscoveredZones[machineType] = zones
	}

	return operation, 0, nil
}
