package provisioning

import (
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	awsTenantName   = "aws-tenant-1"
	azureTenantName = "azure-tenant-2"

	awsEUAccessClaimedSecretName   = "aws-euaccess-tenant-1"
	azureEUAccessClaimedSecretName = "azure-euaccess-tenant-2"
	azureUnclaimedSecretName       = "azure-unclaimed"
	gcpEUAccessSharedSecretName    = "gcp-euaccess-shared"
	awsMostUsedSharedSecretName    = "aws-most-used-shared"
	awsLeastUsedSharedSecretName   = "aws-least-used-shared"
)

func TestResolveSubscriptionSecretStep(t *testing.T) {
	// given
	operationsStorage := storage.NewMemoryStorage().Operations()
	gardenerClient := createGardenerClient()
	rulesService := createRulesService(t)
	stepRetryTuple := internal.RetryTuple{
		Timeout:  2 * time.Second,
		Interval: 1 * time.Second,
	}
	immediateTimeout := internal.RetryTuple{
		Timeout:  -1 * time.Second,
		Interval: 1 * time.Second,
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	t.Run("should resolve secret name for aws hyperscaler and existing tenant", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-1"
			instanceID     = "instance-1"
			platformRegion = "cf-eu11"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.AWS)
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = awsTenantName
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, awsEUAccessClaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)
	})

	t.Run("should resolve secret name for azure hyperscaler and existing tenant", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-2"
			instanceID     = "instance-2"
			platformRegion = "cf-ch20"
			providerType   = "azure"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.Azure)
		operation.ProvisioningParameters.PlanID = broker.AzurePlanID
		operation.ProvisioningParameters.ErsContext.GlobalAccountID = azureTenantName
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, azureEUAccessClaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)
	})

	t.Run("should resolve unclaimed secret name for azure hyperscaler", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-3"
			instanceID     = "instance-3"
			platformRegion = "cf-ap21"
			providerType   = "azure"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.Azure)
		operation.ProvisioningParameters.PlanID = broker.AzurePlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, azureUnclaimedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)
	})

	t.Run("should resolve shared secret name for gcp hyperscaler", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-4"
			instanceID     = "instance-4"
			platformRegion = "cf-eu30"
			providerType   = "gcp"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.GCP)
		operation.ProvisioningParameters.PlanID = broker.GCPPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, gcpEUAccessSharedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)
	})

	t.Run("should resolve the least used shared secret name for aws hyperscaler and trial plan", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-5"
			instanceID     = "instance-5"
			platformRegion = "cf-eu10"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.AWS)
		operation.ProvisioningParameters.PlanID = broker.TrialPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, stepRetryTuple)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.NoError(t, err)
		assert.Zero(t, backoff)
		assert.Equal(t, awsLeastUsedSharedSecretName, *operation.ProvisioningParameters.Parameters.TargetSecret)
	})

	t.Run("should return error on missing rule match for given provisioning attributes", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-6"
			instanceID     = "instance-6"
			platformRegion = "non-existent-region"
			providerType   = "openstack"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.SapConvergedCloud)
		operation.ProvisioningParameters.PlanID = broker.SapConvergedCloudPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, immediateTimeout)

		// when
		_, backoff, err := step.Run(operation, log)

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.True(t, strings.Contains(err.Error(), "no matching rule for provisioning attributes"))
	})

	t.Run("should return error on missing secret binding for given selector", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-7"
			instanceID     = "instance-7"
			platformRegion = "cf-ap11"
			providerType   = "aws"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.AWS)
		operation.ProvisioningParameters.PlanID = broker.AWSPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, immediateTimeout)

		// when
		_, backoff, err := step.Run(operation, log)

		// then
		assert.Error(t, err)
		assert.Zero(t, backoff)
		assert.True(t, strings.Contains(err.Error(), "failed to find unassigned secret binding with selector"))
	})

	t.Run("should fail operation when target secret name is empty", func(t *testing.T) {
		// given
		const (
			operationName  = "provisioning-operation-8"
			instanceID     = "instance-8"
			platformRegion = "cf-us30"
			providerType   = "gcp"
		)

		operation := fixture.FixProvisioningOperationWithProvider(operationName, instanceID, pkg.GCP)
		operation.ProvisioningParameters.PlanID = broker.GCPPlanID
		operation.ProvisioningParameters.PlatformRegion = platformRegion
		operation.ProviderValues = &internal.ProviderValues{ProviderType: providerType}
		require.NoError(t, operationsStorage.InsertOperation(operation))

		step := NewResolveSubscriptionSecretStep(operationsStorage, gardenerClient, rulesService, immediateTimeout)

		// when
		operation, backoff, err := step.Run(operation, log)

		// then
		require.Error(t, err)
		assert.Zero(t, backoff)
		assert.ErrorContains(t, err, "failed to determine secret name")
		assert.Equal(t, domain.Failed, operation.State)
	})
}

func createGardenerClient() *gardener.Client {
	const (
		namespace          = "test"
		secretBindingName1 = "secret-binding-1"
		secretBindingName2 = "secret-binding-2"
		secretBindingName3 = "secret-binding-3"
		secretBindingName4 = "secret-binding-4"
		secretBindingName5 = "secret-binding-5"
		secretBindingName6 = "secret-binding-6"
		secretBindingName7 = "secret-binding-7"
	)
	sb1 := createSecretBinding(secretBindingName1, namespace, awsEUAccessClaimedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      awsTenantName,
	})
	sb2 := createSecretBinding(secretBindingName2, namespace, azureEUAccessClaimedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      azureTenantName,
	})
	sb3 := createSecretBinding(secretBindingName3, namespace, azureUnclaimedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
	})
	sb4 := createSecretBinding(secretBindingName4, namespace, gcpEUAccessSharedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
		gardener.EUAccessLabelKey:        "true",
		gardener.SharedLabelKey:          "true",
	})
	sb5 := createSecretBinding(secretBindingName5, namespace, awsMostUsedSharedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb6 := createSecretBinding(secretBindingName6, namespace, awsLeastUsedSharedSecretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb7 := createSecretBinding(secretBindingName7, namespace, "", map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
	})
	shoot1 := createShoot("shoot-1", namespace, secretBindingName5)
	shoot2 := createShoot("shoot-2", namespace, secretBindingName5)
	shoot3 := createShoot("shoot-3", namespace, secretBindingName6)

	fakeGardenerClient := gardener.NewDynamicFakeClient(sb1, sb2, sb3, sb4, sb5, sb6, sb7, shoot1, shoot2, shoot3)

	return gardener.NewClient(fakeGardenerClient, namespace)
}

func createSecretBinding(name, namespace, secretName string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"secretRef": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
			},
		},
	}
	u.SetLabels(labels)
	u.SetGroupVersionKind(gardener.SecretBindingGVK)

	return u
}

func createShoot(name, namespace, secretBindingName string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"secretBindingName": secretBindingName,
			},
			"status": map[string]interface{}{
				"lastOperation": map[string]interface{}{
					"state": "Succeeded",
					"type":  "Reconcile",
				},
			},
		},
	}
	u.SetGroupVersionKind(gardener.ShootGVK)

	return u
}

func createRulesService(t *testing.T) *rules.RulesService {
	content := `rule:
                      - aws(PR=cf-eu11) -> EU
                      - aws(PR=cf-ap11)
                      - azure(PR=cf-ch20) -> EU
                      - azure(PR=cf-ap21)
                      - gcp(PR=cf-eu30) -> EU,S
                      - gcp(PR=cf-us30)
                      - trial -> S`
	tmpfile, err := rules.CreateTempFile(content)
	require.NoError(t, err)
	defer os.Remove(tmpfile)

	enabledPlans := &broker.EnablePlans{"aws", "azure", "gcp", "trial"}
	rs, err := rules.NewRulesServiceFromFile(tmpfile, enabledPlans)
	require.NoError(t, err)

	return rs
}
