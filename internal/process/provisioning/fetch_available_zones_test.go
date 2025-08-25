package provisioning

import (
	"context"
	"fmt"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFetchAvailableZonesStep_SkipWhenPlatformProviderNotAWS(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewFetchAvailableZonesStep(memoryStorage.Operations(), createGardenerClient(), NewFakeClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestFetchAvailableZonesStep_SkipWhenNoAdditionalWorkerNodePools(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewFetchAvailableZonesStep(memoryStorage.Operations(), createGardenerClient(), NewFakeClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestFetchAvailableZonesStep_FailWhenNoTargetSecret(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:           "worker-1",
		MachineType:    "g6.xlarge",
		HAZones:        false,
		AutoScalerMin:  1,
		AutoScalerMax:  1,
		AvailableZones: []string{},
	}}

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewFetchAvailableZonesStep(memoryStorage.Operations(), createGardenerClient(), NewFakeClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.EqualError(t, err, "target secret is missing")
	assert.Zero(t, repeat)
}

func TestFetchAvailableZonesStep_FailWhenNoRegion(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:           "worker-1",
		MachineType:    "g6.xlarge",
		HAZones:        false,
		AutoScalerMin:  1,
		AutoScalerMax:  1,
		AvailableZones: []string{},
	}}
	targetSecret := "secret-1"
	operation.ProvisioningParameters.Parameters.TargetSecret = &targetSecret
	operation.ProvisioningParameters.Parameters.Region = nil

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewFetchAvailableZonesStep(memoryStorage.Operations(), createGardenerClient(), NewFakeClientFactory(map[string][]string{}, nil))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.EqualError(t, err, "region is missing")
	assert.Zero(t, repeat)
}

func TestExtractAWSCredentials(t *testing.T) {
	testCases := []struct {
		name            string
		unstructured    *unstructured.Unstructured
		error           error
		accessKeyID     string
		secretAccessKey string
	}{
		{
			name: "no data",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			error: fmt.Errorf("secret does not contain data"),
		},
		{
			name: "no accessKeyID",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"secretAccessKey": "dGVzdC1zZWNyZXQ=",
					},
				},
			},
			error: fmt.Errorf("secret does not contain accessKeyID"),
		},
		{
			name: "no secretAccessKey",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID": "dGVzdC1rZXk=",
					},
				},
			},
			error: fmt.Errorf("secret does not contain secretAccessKey"),
		},
		{
			name: "invalid accessKeyID",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID":     "test-key",
						"secretAccessKey": "dGVzdC1zZWNyZXQ=",
					},
				},
			},
			error: fmt.Errorf("failed to decode accessKeyID"),
		},
		{
			name: "invalid secretAccessKey",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID":     "dGVzdC1rZXk=",
						"secretAccessKey": "test-secret",
					},
				},
			},
			error: fmt.Errorf("failed to decode secretAccessKey"),
		},
		{
			name: "valid credentials",
			unstructured: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"data": map[string]interface{}{
						"accessKeyID":     "dGVzdC1rZXk=",
						"secretAccessKey": "dGVzdC1zZWNyZXQ=",
					},
				},
			},
			accessKeyID:     "test-key",
			secretAccessKey: "test-secret",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			step := NewFetchAvailableZonesStep(storage.NewMemoryStorage().Operations(), createGardenerClient(), NewFakeClientFactory(map[string][]string{}, nil))

			// when
			accessKeyID, secretAccessKey, err := step.extractAWSCredentials(tc.unstructured)

			// then
			if tc.error != nil {
				assert.Contains(t, err.Error(), tc.error.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.accessKeyID, accessKeyID)
				assert.Equal(t, tc.secretAccessKey, secretAccessKey)
			}
		})
	}
}

func TestFetchAvailableZonesStep_RepeatWhenAWSError(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{{
		Name:           "worker-1",
		MachineType:    "g6.xlarge",
		HAZones:        false,
		AutoScalerMin:  1,
		AutoScalerMax:  1,
		AvailableZones: []string{},
	}}
	targetSecret := "secret-1"
	operation.ProvisioningParameters.Parameters.TargetSecret = &targetSecret

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewFetchAvailableZonesStep(memoryStorage.Operations(), createGardenerClient(), NewFakeClientFactory(map[string][]string{}, fmt.Errorf("AWS error")))

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
}

func TestFetchAvailableZonesStep_HappyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	instance := fixInstance()
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.RuntimeID = instance.RuntimeID
	operation.ProvisioningParameters.PlatformProvider = pkg.AWS
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:           "worker-1",
			MachineType:    "g6.xlarge",
			HAZones:        false,
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			AvailableZones: []string{},
		},
		{
			Name:           "worker-2",
			MachineType:    "g4dn.xlarge",
			HAZones:        false,
			AutoScalerMin:  1,
			AutoScalerMax:  1,
			AvailableZones: []string{},
		},
	}
	targetSecret := "secret-1"
	operation.ProvisioningParameters.Parameters.TargetSecret = &targetSecret

	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	step := NewFetchAvailableZonesStep(
		memoryStorage.Operations(),
		createGardenerClient(),
		NewFakeClientFactory(map[string][]string{
			"g6.xlarge":   {"ap-southeast-2a", "ap-southeast-2c"},
			"g4dn.xlarge": {"ap-southeast-2b"},
		}, nil),
	)

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	updatedOperation, err := memoryStorage.Operations().GetOperationByID(operationID)
	assert.NoError(t, err)
	require.Len(t, updatedOperation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools, 2)
	assert.ElementsMatch(t, updatedOperation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools[0].AvailableZones, []string{"ap-southeast-2a", "ap-southeast-2c"})
	assert.ElementsMatch(t, updatedOperation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools[1].AvailableZones, []string{"ap-southeast-2b"})
}

func NewFakeClientFactory(zones map[string][]string, error error) *FakeClientFactory {
	fakeClient := &fakeClient{
		zones: zones,
		err:   error,
	}
	return &FakeClientFactory{client: fakeClient}
}

type FakeClientFactory struct {
	client aws.Client
}

func (f *FakeClientFactory) New(ctx context.Context, accessKeyID, secretAccessKey, region string) (aws.Client, error) {
	return f.client, nil
}

type fakeClient struct {
	zones map[string][]string
	err   error
}

func (f *fakeClient) AvailableZones(ctx context.Context, machineType string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.zones[machineType], nil
}
