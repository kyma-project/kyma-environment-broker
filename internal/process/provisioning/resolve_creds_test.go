package provisioning

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/pivotal-cf/brokerapi/v12/domain"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	hyperscalerMocks "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

const (
	namespace = "kyma-dev"
	tenant    = "tenant"
)

func TestResolveCredentialsStepHappyPath_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, pkg.GCP)
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSecretName", hyperscaler.GCP("westeurope"), statusGlobalAccountID, false).Return("gardener-secret-gcp", nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock, &rules.RulesService{})

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-gcp", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsEUStepHappyPath_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.AWSPlanID, pkg.AWS)
	operation.ProvisioningParameters.PlatformRegion = "cf-eu11"
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSecretName", hyperscaler.AWS(), statusGlobalAccountID, true).Return("gardener-secret-aws", nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock, &rules.RulesService{})

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-aws", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsCHStepHappyPath_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.AWSPlanID, pkg.Azure)
	operation.ProvisioningParameters.PlatformRegion = "cf-ch20"
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSecretName", hyperscaler.Azure(), statusGlobalAccountID, true).Return("gardener-secret-az", nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock, &rules.RulesService{})

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-az", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsStepHappyPathTrialDefaultProvider_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.TrialPlanID, pkg.Azure)
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSharedSecretName", hyperscaler.Azure(), false).Return("gardener-secret-azure", nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock, &rules.RulesService{})

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-azure", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsStepHappyPathTrialGivenProvider_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatusWithProvider(broker.TrialPlanID, pkg.GCP)

	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSharedSecretName", hyperscaler.GCP("westeurope"), false).Return("gardener-secret-gcp", nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock, &rules.RulesService{})

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Empty(t, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-gcp", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsStepRetry_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, pkg.GCP)
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSecretName", hyperscaler.GCP("westeurope"), statusGlobalAccountID, false).Return("", fmt.Errorf("Failed!"))

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock, &rules.RulesService{})

	operation.UpdatedAt = time.Now()

	// when
	operation, repeat, err := step.Run(operation, fixLogger())

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
	assert.Nil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, domain.InProgress, operation.State)

	operation, repeat, err = step.Run(operation, fixLogger())

	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	assert.Nil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentials_IntegrationAWS(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	gc := gardener.NewDynamicFakeClient(
		fixSecretBinding("s1aws", "aws"),
		fixSecretBinding("s1azure", "azure"))
	accountProvider := hyperscaler.NewAccountProvider(hyperscaler.NewAccountPool(gc, namespace), hyperscaler.NewSharedGardenerAccountPool(gc, namespace))

	op := fixOperationWithPlatformRegion("cf-us10", pkg.AWS)
	err := memoryStorage.Operations().InsertOperation(op)
	assert.NoError(t, err)
	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProvider, &rules.RulesService{})

	// when
	operation, backoff, err := step.Run(op, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assert.Equal(t, "s1aws", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentials_IntegrationAWSEuAccess(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	gc := gardener.NewDynamicFakeClient(
		fixSecretBinding("aws", "aws"),
		fixSecretBinding("azure", "azure"),
		fixEuAccessSecretBinding("awseu", "aws"),
		fixEuAccessSecretBinding("azureeu", "azure"))
	accountProvider := hyperscaler.NewAccountProvider(hyperscaler.NewAccountPool(gc, namespace), hyperscaler.NewSharedGardenerAccountPool(gc, namespace))

	op := fixOperationWithPlatformRegion("cf-eu11", pkg.AWS)
	err := memoryStorage.Operations().InsertOperation(op)
	assert.NoError(t, err)
	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProvider, &rules.RulesService{})

	// when
	operation, backoff, err := step.Run(op, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assert.Equal(t, "awseu", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentials_IntegrationAzure(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	gc := gardener.NewDynamicFakeClient(
		fixSecretBinding("s1aws", "aws"),
		fixSecretBinding("s1azure", "azure"))
	accountProvider := hyperscaler.NewAccountProvider(hyperscaler.NewAccountPool(gc, namespace), hyperscaler.NewSharedGardenerAccountPool(gc, namespace))

	op := fixOperationWithPlatformRegion("cf-eu21", pkg.Azure)
	err := memoryStorage.Operations().InsertOperation(op)
	assert.NoError(t, err)
	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProvider, &rules.RulesService{})

	// when
	operation, backoff, err := step.Run(op, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assert.Equal(t, "s1azure", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentials_IntegrationAzureEuAccess(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	gc := gardener.NewDynamicFakeClient(
		fixSecretBinding("aws", "aws"),
		fixSecretBinding("azure", "azure"),
		fixEuAccessSecretBinding("awseu", "aws"),
		fixEuAccessSecretBinding("azureeu", "azure"))
	accountProvider := hyperscaler.NewAccountProvider(hyperscaler.NewAccountPool(gc, namespace), hyperscaler.NewSharedGardenerAccountPool(gc, namespace))

	op := fixOperationWithPlatformRegion("cf-ch20", pkg.Azure)
	err := memoryStorage.Operations().InsertOperation(op)
	assert.NoError(t, err)
	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProvider, &rules.RulesService{})

	// when
	operation, backoff, err := step.Run(op, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	assert.Equal(t, "azureeu", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func fixOperationWithPlatformRegion(platformRegion string, provider pkg.CloudProvider) internal.Operation {
	o := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	o.ProvisioningParameters.PlatformRegion = platformRegion

	return o
}

var gvk = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "SecretBinding"}

func fixSecretBinding(name, hyperscalerType string) *unstructured.Unstructured {
	o := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"hyperscalerType": hyperscalerType,
				},
			},
			"secretRef": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	o.SetGroupVersionKind(gvk)
	return o
}

func fixEuAccessSecretBinding(name, hyperscalerType string) *unstructured.Unstructured {
	o := fixSecretBinding(name, hyperscalerType)
	labels := o.GetLabels()
	labels["euAccess"] = "true"
	o.SetLabels(labels)
	return o
}
