package provisioning

import (
	"fmt"
	"strings"
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
	namespace                    = "kyma-dev"
	tenant                       = "tenant"
	statusOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	statusInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	statusRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	statusGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	statusProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"

	dashboardURL = "http://runtime.com"
)

func TestResolveCredentialsStepHappyPath_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, pkg.GCP)
	operation.ProviderValues = &internal.ProviderValues{
		ProviderType: "gcp",
	}
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
	op.ProviderValues = &internal.ProviderValues{
		ProviderType: "aws",
	}
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
	op.ProviderValues = &internal.ProviderValues{
		ProviderType: "aws",
	}
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
	op.ProviderValues = &internal.ProviderValues{
		ProviderType: "azure",
	}
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
	op.ProviderValues = &internal.ProviderValues{
		ProviderType: "azure",
	}
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

func fixOperationRuntimeStatus(planId string, provider pkg.CloudProvider) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	provisioningOperation.State = domain.InProgress
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.InstanceDetails.RuntimeID = runtimeID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID
	provisioningOperation.ProviderValues = &internal.ProviderValues{
		ProviderType: strings.ToLower(string(provider)),
	}

	return provisioningOperation
}

func fixOperationRuntimeStatusWithProvider(planId string, provider pkg.CloudProvider) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	provisioningOperation.State = ""
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID
	provisioningOperation.ProvisioningParameters.Parameters.Provider = &provider
	provisioningOperation.ProviderValues = &internal.ProviderValues{
		ProviderType: strings.ToLower(string(provider)),
	}
	return provisioningOperation
}
