package steps

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
)

const (
	instanceID  = "instance-1"
	operationID = "operation-1"
)

func TestDiscoverAvailableZonesCBStep_ZonesDiscoveryDisabled(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = "aws-most-used-shared"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{ProviderType: "aws"}
	operation.RuntimeID = instance.RuntimeID
	machineType := "m6i.large"
	operation.ProvisioningParameters.Parameters.MachineType = &machineType
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "worker-1",
			MachineType:   "g6.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
	}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(
		memoryStorage,
		fixture.NewProviderSpecWithZonesDiscovery(t, false),
		fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{
			"m6i.large":   {"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"},
			"g6.xlarge":   {"ap-southeast-2a", "ap-southeast-2c"},
			"g4dn.xlarge": {"ap-southeast-2b"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestDiscoverAvailableZonesCBStep_FailWhenNoSubscriptionSecretName(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = ""
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{ProviderType: "aws"}
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:          "worker-1",
		MachineType:   "g6.xlarge",
		HAZones:       false,
		AutoScalerMin: 1,
		AutoScalerMax: 1,
	}}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(memoryStorage, fixture.NewProviderSpecWithZonesDiscovery(t, true), fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.EqualError(t, err, "subscription secret name is missing")
	assert.Zero(t, repeat)
}

func TestDiscoverAvailableZonesCBStep_SubscriptionSecretNameFromOperation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = ""
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{ProviderType: "aws"}
	operation.RuntimeID = instance.RuntimeID
	machineType := "m6i.large"
	operation.ProvisioningParameters.Parameters.MachineType = &machineType
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "worker-1",
			MachineType:   "g6.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
		{
			Name:          "worker-2",
			MachineType:   "g4dn.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
	}
	subscriptionSecretName := "aws-most-used-shared"
	operation.ProvisioningParameters.Parameters.TargetSecret = &subscriptionSecretName
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(
		memoryStorage,
		fixture.NewProviderSpecWithZonesDiscovery(t, true),
		fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{
			"m6i.large":   {"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"},
			"g6.xlarge":   {"ap-southeast-2a", "ap-southeast-2c"},
			"g4dn.xlarge": {"ap-southeast-2b"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	assert.Len(t, operation.DiscoveredZones, 3)
	assert.ElementsMatch(t, operation.DiscoveredZones["m6i.large"], []string{"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g6.xlarge"], []string{"ap-southeast-2a", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g4dn.xlarge"], []string{"ap-southeast-2b"})
}

func TestDiscoverAvailableZonesCBStep_RegionFromProviderValues(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = "aws-most-used-shared"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.Parameters.Region = nil
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{Region: "eu-west-2", ProviderType: "aws"}
	machineType := "m6i.large"
	operation.ProvisioningParameters.Parameters.MachineType = &machineType
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "worker-1",
			MachineType:   "g6.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
		{
			Name:          "worker-2",
			MachineType:   "g4dn.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
	}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(
		memoryStorage,
		fixture.NewProviderSpecWithZonesDiscovery(t, true),
		fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{
			"m6i.large":   {"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"},
			"g6.xlarge":   {"ap-southeast-2a", "ap-southeast-2c"},
			"g4dn.xlarge": {"ap-southeast-2b"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	assert.Len(t, operation.DiscoveredZones, 3)
	assert.ElementsMatch(t, operation.DiscoveredZones["m6i.large"], []string{"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g6.xlarge"], []string{"ap-southeast-2a", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g4dn.xlarge"], []string{"ap-southeast-2b"})
}

func TestDiscoverAvailableZonesCBStep_MachineTypeFromProviderValues(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = "aws-most-used-shared"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.Parameters.MachineType = nil
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{DefaultMachineType: "m5.large", ProviderType: "aws"}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(
		memoryStorage,
		fixture.NewProviderSpecWithZonesDiscovery(t, true),
		fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{
			"m5.large": {"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	assert.Len(t, operation.DiscoveredZones, 1)
	assert.ElementsMatch(t, operation.DiscoveredZones["m5.large"], []string{"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"})
}

func TestDiscoverAvailableZonesCBStep_RepeatWhenAWSError(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = "aws-most-used-shared"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{ProviderType: "aws"}
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:          "worker-1",
		MachineType:   "g6.xlarge",
		HAZones:       false,
		AutoScalerMin: 1,
		AutoScalerMax: 1,
	}}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(memoryStorage, fixture.NewProviderSpecWithZonesDiscovery(t, true),
		fixture.CreateGardenerClientWithCredentialsBindings(), fixture.NewFakeAWSClientFactory(map[string][]string{}, fmt.Errorf("AWS error")))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
}

func TestDiscoverAvailableZonesCBStep_ProvisioningHappyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = "aws-most-used-shared"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{ProviderType: "aws"}
	operation.RuntimeID = instance.RuntimeID
	machineType := "m6i.large"
	operation.ProvisioningParameters.Parameters.MachineType = &machineType
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "worker-1",
			MachineType:   "g6.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
		{
			Name:          "worker-2",
			MachineType:   "g4dn.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
	}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(
		memoryStorage,
		fixture.NewProviderSpecWithZonesDiscovery(t, true),
		fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{
			"m6i.large":   {"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"},
			"g6.xlarge":   {"ap-southeast-2a", "ap-southeast-2c"},
			"g4dn.xlarge": {"ap-southeast-2b"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	assert.Len(t, operation.DiscoveredZones, 3)
	assert.ElementsMatch(t, operation.DiscoveredZones["m6i.large"], []string{"ap-southeast-2a", "ap-southeast-2b", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g6.xlarge"], []string{"ap-southeast-2a", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g4dn.xlarge"], []string{"ap-southeast-2b"})
}

func TestDiscoverAvailableZonesCBStep_UpdateHappyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixture.FixInstance(instanceID)
	instance.SubscriptionSecretName = "aws-most-used-shared"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixUpdatingOperation(operationID, instanceID).Operation
	operation.InstanceDetails.ProviderValues = &internal.ProviderValues{ProviderType: "aws"}
	operation.RuntimeID = instance.RuntimeID
	operation.UpdatingParameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "worker-1",
			MachineType:   "g6.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
		{
			Name:          "worker-2",
			MachineType:   "g4dn.xlarge",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
	}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesCBStep(
		memoryStorage,
		fixture.NewProviderSpecWithZonesDiscovery(t, true),
		fixture.CreateGardenerClientWithCredentialsBindings(),
		fixture.NewFakeAWSClientFactory(map[string][]string{
			"g6.xlarge":   {"ap-southeast-2a", "ap-southeast-2c"},
			"g4dn.xlarge": {"ap-southeast-2b"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	assert.Len(t, operation.DiscoveredZones, 2)
	assert.ElementsMatch(t, operation.DiscoveredZones["g6.xlarge"], []string{"ap-southeast-2a", "ap-southeast-2c"})
	assert.ElementsMatch(t, operation.DiscoveredZones["g4dn.xlarge"], []string{"ap-southeast-2b"})
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
