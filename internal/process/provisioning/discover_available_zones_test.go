package provisioning

import (
	"fmt"
	"strings"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverAvailableZonesStep_ZonesDiscoveryDisabled(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	instance.SubscriptionSecretName = "secret-1"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
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

	step := NewDiscoverAvailableZonesStep(
		memoryStorage,
		newProviderSpec(t, false),
		createGardenerClient(),
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

func TestDiscoverAvailableZonesStep_FailWhenNoRegion(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	instance.SubscriptionSecretName = "secret-1"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:          "worker-1",
		MachineType:   "g6.xlarge",
		HAZones:       false,
		AutoScalerMin: 1,
		AutoScalerMax: 1,
	}}
	operation.ProvisioningParameters.Parameters.Region = nil
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesStep(memoryStorage, newProviderSpec(t, true), createGardenerClient(), fixture.NewFakeAWSClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.EqualError(t, err, "region is missing")
	assert.Zero(t, repeat)
}

func TestDiscoverAvailableZonesStep_FailWhenNoSubscriptionSecretName(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:          "worker-1",
		MachineType:   "g6.xlarge",
		HAZones:       false,
		AutoScalerMin: 1,
		AutoScalerMax: 1,
	}}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesStep(memoryStorage, newProviderSpec(t, true), createGardenerClient(), fixture.NewFakeAWSClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.EqualError(t, err, "subscription secret name is missing")
	assert.Zero(t, repeat)
}

func TestDiscoverAvailableZonesStep_RepeatWhenAWSError(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	instance.SubscriptionSecretName = "secret-1"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:          "worker-1",
		MachineType:   "g6.xlarge",
		HAZones:       false,
		AutoScalerMin: 1,
		AutoScalerMax: 1,
	}}
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesStep(memoryStorage, newProviderSpec(t, true), createGardenerClient(), fixture.NewFakeAWSClientFactory(map[string][]string{}, fmt.Errorf("AWS error")))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
}

func TestDiscoverAvailableZonesStep_ProvisioningHappyPath_SubscriptionSecretNameFromInstance(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	instance.SubscriptionSecretName = "secret-1"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
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

	step := NewDiscoverAvailableZonesStep(
		memoryStorage,
		newProviderSpec(t, true),
		createGardenerClient(),
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

func TestDiscoverAvailableZonesStep_ProvisioningHappyPath_SubscriptionSecretNameFromOperation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
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
	subscriptionSecretName := "secret-1"
	operation.ProvisioningParameters.Parameters.TargetSecret = &subscriptionSecretName
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewDiscoverAvailableZonesStep(
		memoryStorage,
		newProviderSpec(t, true),
		createGardenerClient(),
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

func TestDiscoverAvailableZonesStep_UpdateHappyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	instance.SubscriptionSecretName = "secret-1"
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixUpdatingOperation(operationID, instanceID).Operation
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
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

	step := NewDiscoverAvailableZonesStep(
		memoryStorage,
		newProviderSpec(t, true),
		createGardenerClient(),
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

func newProviderSpec(t *testing.T, zonesDiscovery bool) *configuration.ProviderSpec {
	spec := fmt.Sprintf(`
aws:
  zonesDiscovery: %t
`, zonesDiscovery)
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(spec))
	require.NoError(t, err)
	return providerSpec
}
