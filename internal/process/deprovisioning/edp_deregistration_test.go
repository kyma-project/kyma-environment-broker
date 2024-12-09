package deprovisioning

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

const (
	edpName        = "f88401ba-c601-45bb-bec0-a2156c07c9a6"
	edpEnvironment = "test"
)

var metadataTenantKeys = []string{
	edp.MaasConsumerEnvironmentKey,
	edp.MaasConsumerRegionKey,
	edp.MaasConsumerSubAccountKey,
	edp.MaasConsumerServicePlan,
}

func TestEDPDeregistration_Run(t *testing.T) {
	// given
	client := edp.NewFakeClient()
	prepareEDP(t, edpName, client)

	memoryStorage := storage.NewMemoryStorage()
	_, operation := prepareDeprovisioningInstanceWithSubaccount(t, edpName, memoryStorage.Instances(), memoryStorage.Operations())

	step := NewEDPDeregistrationStep(memoryStorage.Operations(), memoryStorage.Instances(), client, edp.Config{
		Environment: edpEnvironment,
	})

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, 0*time.Second, repeat)
	assert.NoError(t, err)

	for _, key := range metadataTenantKeys {
		metadataTenant, metadataTenantExists := client.GetMetadataItem(edpName, edpEnvironment, key)
		assert.False(t, metadataTenantExists)
		assert.Equal(t, edp.MetadataItem{}, metadataTenant)
	}

	dataTenant, dataTenantExists := client.GetDataTenantItem(edpName, edpEnvironment)
	assert.False(t, dataTenantExists)
	assert.Equal(t, edp.DataTenantItem{}, dataTenant)
}

func TestEDPDeregistration_RunWithOtherInstances(t *testing.T) {
	// given
	client := edp.NewFakeClient()
	prepareEDP(t, edpName, client)

	memoryStorage := storage.NewMemoryStorage()
	_, _ = prepareProvisionedInstanceWithSubaccount(t, edpName, memoryStorage.Instances(), memoryStorage.Operations())
	_, operation := prepareDeprovisioningInstanceWithSubaccount(t, edpName, memoryStorage.Instances(), memoryStorage.Operations())

	step := NewEDPDeregistrationStep(memoryStorage.Operations(), memoryStorage.Instances(), client, edp.Config{
		Environment: edpEnvironment,
	})

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, 0*time.Second, repeat)
	assert.NoError(t, err)

	for _, key := range metadataTenantKeys {
		_, metadataTenantExists := client.GetMetadataItem(edpName, edpEnvironment, key)
		assert.True(t, metadataTenantExists)
	}

	_, dataTenantExists := client.GetDataTenantItem(edpName, edpEnvironment)
	assert.True(t, dataTenantExists)
}

func TestEDPDeregistration_RunWithOtherInstancesButDifferentSubaccount(t *testing.T) {
	// given
	client := edp.NewFakeClient()
	prepareEDP(t, edpName, client)

	memoryStorage := storage.NewMemoryStorage()
	_, _ = prepareProvisionedInstanceWithSubaccount(t, "subaccount-other", memoryStorage.Instances(), memoryStorage.Operations())
	_, operation := prepareDeprovisioningInstanceWithSubaccount(t, edpName, memoryStorage.Instances(), memoryStorage.Operations())

	step := NewEDPDeregistrationStep(memoryStorage.Operations(), memoryStorage.Instances(), client, edp.Config{
		Environment: edpEnvironment,
	})

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, 0*time.Second, repeat)
	assert.NoError(t, err)

	for _, key := range metadataTenantKeys {
		_, metadataTenantExists := client.GetMetadataItem(edpName, edpEnvironment, key)
		assert.False(t, metadataTenantExists)
	}

	_, dataTenantExists := client.GetDataTenantItem(edpName, edpEnvironment)
	assert.False(t, dataTenantExists)
}

func TestEDPDeregistration_RunWithOtherInstancesInDeprovisioningState(t *testing.T) {
	// given
	client := edp.NewFakeClient()
	prepareEDP(t, edpName, client)

	memoryStorage := storage.NewMemoryStorage()
	_, _ = prepareDeprovisioningInstanceWithSubaccount(t, edpName, memoryStorage.Instances(), memoryStorage.Operations())
	_, operation := prepareDeprovisioningInstanceWithSubaccount(t, edpName, memoryStorage.Instances(), memoryStorage.Operations())

	step := NewEDPDeregistrationStep(memoryStorage.Operations(), memoryStorage.Instances(), client, edp.Config{
		Environment: edpEnvironment,
	})

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, 0*time.Second, repeat)
	assert.NoError(t, err)

	for _, key := range metadataTenantKeys {
		_, metadataTenantExists := client.GetMetadataItem(edpName, edpEnvironment, key)
		assert.False(t, metadataTenantExists)
	}

	_, dataTenantExists := client.GetDataTenantItem(edpName, edpEnvironment)
	assert.False(t, dataTenantExists)
}

func prepareEDP(t *testing.T, subaccountId string, client *edp.FakeClient) {
	err := client.CreateDataTenant(edp.DataTenantPayload{
		Name:        subaccountId,
		Environment: edpEnvironment,
		Secret:      base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", edpName, edpEnvironment))),
	}, fixLogger())
	assert.NoError(t, err)

	for _, key := range metadataTenantKeys {
		err = client.CreateMetadataTenant(subaccountId, edpEnvironment, edp.MetadataTenantPayload{
			Key:   key,
			Value: "-",
		}, fixLogger())
		assert.NoError(t, err)
	}
}

func prepareProvisionedInstanceWithSubaccount(t *testing.T, subaccountId string, instances storage.Instances, operations storage.Operations) (internal.Instance, internal.Operation) {
	instanceID := uuid.New().String()
	instance := fixture.FixInstance(instanceID)
	instance.SubAccountID = subaccountId
	operation := fixture.FixProvisioningOperation(uuid.New().String(), instanceID)
	operation.SubAccountID = subaccountId
	err := instances.Insert(instance)
	assert.NoError(t, err)
	err = operations.InsertOperation(operation)
	assert.NoError(t, err)

	return instance, operation
}

func prepareDeprovisioningInstanceWithSubaccount(t *testing.T, subaccountId string, instances storage.Instances, operations storage.Operations) (internal.Instance, internal.Operation) {
	instanceID := uuid.New().String()
	instance := fixture.FixInstance(instanceID)
	instance.SubAccountID = subaccountId
	operation := fixture.FixDeprovisioningOperationAsOperation(uuid.New().String(), instanceID)
	operation.SubAccountID = subaccountId
	err := instances.Insert(instance)
	assert.NoError(t, err)
	err = operations.InsertOperation(operation)
	assert.NoError(t, err)

	return instance, operation
}
