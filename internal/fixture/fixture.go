package fixture

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceId                      = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	ServiceName                    = "kymaruntime"
	PlanId                         = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	TrialPlan                      = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	PlanName                       = "azure"
	SubscriptionSecretName         = "azure-subscription"
	GlobalAccountId                = "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4"
	SubscriptionGlobalAccountID    = "subscription-global-account-id"
	Region                         = "westeurope"
	InstanceDashboardURL           = "https://dashboard.local"
	MonitoringUsername             = "username"
	MonitoringPassword             = "password"
	AWSTenantName                  = "aws-tenant-1"
	AzureTenantName                = "azure-tenant-2"
	AWSSecretName                  = "aws-secret"
	AWSSecretName2                 = "aws-secret-2"
	AWSSecretName3                 = "aws-secret-3"
	AWSEUAccessClaimedSecretName   = "aws-euaccess-tenant-1"
	AzureEUAccessClaimedSecretName = "azure-euaccess-tenant-2"
	AzureUnclaimedSecretName       = "azure-unclaimed"
	GCPEUAccessSharedSecretName    = "gcp-euaccess-shared"
	AWSMostUsedSharedSecretName    = "aws-most-used-shared"
	AWSLeastUsedSharedSecretName   = "aws-least-used-shared"
)

func FixERSContext(id string) internal.ERSContext {
	var (
		tenantID     = fmt.Sprintf("Tenant-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
		userID       = fmt.Sprintf("User-%s", id)
		licenseType  = "SAPDEV"
	)

	return internal.ERSContext{
		TenantID:        tenantID,
		SubAccountID:    subAccountId,
		GlobalAccountID: GlobalAccountId,
		Active:          ptr.Bool(true),
		UserID:          userID,
		LicenseType:     &licenseType,
	}
}

func FixDNSProvidersConfig() gardener.DNSProvidersData {
	return gardener.DNSProvidersData{
		Providers: []gardener.DNSProviderData{
			{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_incustom",
				Type:           "route53_type_test",
			},
		},
	}
}

const KymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma
  namespace: kyma-system
spec:
  sync:
    strategy: secret
  channel: fast
  modules: []
`

type FakeKymaConfigProvider struct{}

func (FakeKymaConfigProvider) Provide(cfgKeyName string, cfgDestObj any) error {
	cfg, _ := cfgDestObj.(*internal.ConfigForPlan)
	cfg.KymaTemplate = KymaTemplate
	cfgDestObj = cfg
	return nil
}

func FixKymaResourceWithGivenRuntimeID(kcpClient client.Client, kymaResourceNamespace string, resourceName string) error {
	return kcpClient.Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operator.kyma-project.io/v1beta2",
		"kind":       "Kyma",
		"metadata": map[string]interface{}{
			"name":      resourceName,
			"namespace": kymaResourceNamespace,
		},
		"spec": map[string]interface{}{
			"channel": "fast",
		},
	}})
}

func NewFakeFactory(zones map[string][]string, err error) *FakeFactory {
	return &FakeFactory{zones: zones, err: err}
}

type FakeFactory struct {
	zones map[string][]string
	err   error
}

func (f *FakeFactory) NewFromSecret(_ context.Context, _ pkg.CloudProvider, _ *unstructured.Unstructured, _ string) (hyperscalers.ProviderClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeProviderClient{zones: f.zones}, nil
}

func (f *FakeFactory) NewPerCallFromSecret(_ context.Context, _ pkg.CloudProvider, _ *unstructured.Unstructured, _ string) (hyperscalers.ProviderClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeProviderClient{zones: f.zones}, nil
}

type fakeProviderClient struct {
	zones map[string][]string
}

func (f *fakeProviderClient) AvailableZones(_ context.Context, machineType string) ([]string, error) {
	return f.zones[machineType], nil
}

func (f *fakeProviderClient) AvailableZonesCount(ctx context.Context, machineType string) (int, error) {
	zones, err := f.AvailableZones(ctx, machineType)
	if err != nil {
		return 0, err
	}
	return len(zones), nil
}

func CreateGardenerClientWithCredentialsBindings() *gardener.Client {
	const (
		namespace   = "test"
		secretName1 = "secret-1"
		secretName2 = "secret-2"
		secretName3 = "secret-3"
		secretName4 = "secret-4"
		secretName5 = "secret-5"
		secretName6 = "secret-6"
		secretName7 = "secret-7"
		secretName8 = "secret-8"
	)
	s1 := createSecret(secretName1, namespace)
	s2 := createSecret(secretName2, namespace)
	s3 := createSecret(secretName3, namespace)
	s4 := createSecret(secretName4, namespace)
	s5 := createSecret(secretName5, namespace)
	s6 := createSecret(secretName6, namespace)
	s7 := createSecret(secretName7, namespace)
	s8 := createSecret(secretName8, namespace)
	sb1 := createCredentialsBinding(AWSEUAccessClaimedSecretName, namespace, secretName1, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb2 := createCredentialsBinding(AzureEUAccessClaimedSecretName, namespace, secretName2, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      AzureTenantName,
	})
	sb3 := createCredentialsBinding(AzureUnclaimedSecretName, namespace, secretName3, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
	})
	sb4 := createCredentialsBinding(GCPEUAccessSharedSecretName, namespace, secretName4, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
		gardener.EUAccessLabelKey:        "true",
		gardener.SharedLabelKey:          "true",
	})
	sb5 := createCredentialsBinding(AWSMostUsedSharedSecretName, namespace, secretName5, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb6 := createCredentialsBinding(AWSLeastUsedSharedSecretName, namespace, secretName6, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb7 := createCredentialsBinding("", namespace, secretName7, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
	})
	sb8 := createCredentialsBinding(AWSSecretName, namespace, secretName8, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	shoot1 := createShoot("shoot-1", namespace, AWSMostUsedSharedSecretName)
	shoot2 := createShoot("shoot-2", namespace, AWSMostUsedSharedSecretName)
	shoot3 := createShoot("shoot-3", namespace, AWSLeastUsedSharedSecretName)

	fakeGardenerClient := gardener.NewDynamicFakeClient(s1, s2, s3, s4, s5, s6, s7, s8, sb1, sb2, sb3, sb4, sb5, sb6, sb7, sb8, shoot1, shoot2, shoot3)

	return gardener.NewClient(fakeGardenerClient, namespace)
}

func CreateGardenerClientWithMultipleAWSBindings() *gardener.Client {
	const (
		namespace   = "test"
		secretName1 = "secret-1"
		secretName2 = "secret-2"
	)
	s1 := createSecret(secretName1, namespace)
	s2 := createSecret(secretName2, namespace)
	sb1 := createCredentialsBinding(AWSSecretName, namespace, secretName1, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb2 := createCredentialsBinding(AWSSecretName2, namespace, secretName2, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})

	fakeGardenerClient := gardener.NewDynamicFakeClient(s1, s2, sb1, sb2)

	return gardener.NewClient(fakeGardenerClient, namespace)
}

func CreateGardenerClientWithThreeAWSBindings() *gardener.Client {
	const (
		namespace   = "test"
		secretName1 = "secret-1"
		secretName2 = "secret-2"
		secretName3 = "secret-3"
	)
	s1 := createSecret(secretName1, namespace)
	s2 := createSecret(secretName2, namespace)
	s3 := createSecret(secretName3, namespace)
	sb1 := createCredentialsBinding(AWSSecretName, namespace, secretName1, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb2 := createCredentialsBinding(AWSSecretName2, namespace, secretName2, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb3 := createCredentialsBinding(AWSSecretName3, namespace, secretName3, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})

	fakeGardenerClient := gardener.NewDynamicFakeClient(s1, s2, s3, sb1, sb2, sb3)

	return gardener.NewClient(fakeGardenerClient, namespace)
}

func CreateGardenerClientWithThreeAWSBindingsAndOneUnclaimed() *gardener.Client {
	const (
		namespace   = "test"
		secretName1 = "secret-1"
		secretName2 = "secret-2"
		secretName3 = "secret-3"
		secretName4 = "secret-4"
	)
	s1 := createSecret(secretName1, namespace)
	s2 := createSecret(secretName2, namespace)
	s3 := createSecret(secretName3, namespace)
	s4 := createSecret(secretName4, namespace)
	sb1 := createCredentialsBinding(AWSSecretName, namespace, secretName1, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb2 := createCredentialsBinding(AWSSecretName2, namespace, secretName2, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb3 := createCredentialsBinding(AWSSecretName3, namespace, secretName3, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	// Unclaimed binding (no tenantName)
	sb4 := createCredentialsBinding("aws-unclaimed", namespace, secretName4, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
	})

	fakeGardenerClient := gardener.NewDynamicFakeClient(s1, s2, s3, s4, sb1, sb2, sb3, sb4)

	return gardener.NewClient(fakeGardenerClient, namespace)
}

func CreateGardenerClientWithAzureCredentialsBindings() *gardener.Client {
	const (
		namespace  = "test"
		secretName = "azure-secret-1"
	)
	secret := createAzureSecret(secretName, namespace)
	binding := createCredentialsBinding(AzureUnclaimedSecretName, namespace, secretName, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
	})
	return gardener.NewClient(gardener.NewDynamicFakeClient(secret, binding), namespace)
}

func createAzureSecret(name, namespace string) *unstructured.Unstructured {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{"name": name, "namespace": namespace},
			"data": map[string]interface{}{
				"clientID":       enc("test-client-id"),
				"clientSecret":   enc("test-client-secret"),
				"tenantID":       enc("test-tenant-id"),
				"subscriptionID": enc("test-subscription-id-12345"),
			},
		},
	}
	u.SetGroupVersionKind(gardener.SecretGVK)
	return u
}

func NewAzureProviderSpecWithZonesDiscovery(t *testing.T) *configuration.ProviderSpec {
	spec := `
azure:
  zonesDiscovery: true
  regions:
    westeurope:
      displayName: "West Europe"
`
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(spec))
	require.NoError(t, err)
	return providerSpec
}

func createSecret(name, namespace string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"data": map[string]interface{}{
				"accessKeyID":     "dGVzdC1rZXk=",
				"secretAccessKey": "dGVzdC1zZWNyZXQ=",
			},
		},
	}
	u.SetGroupVersionKind(gardener.SecretGVK)

	return u
}

func createCredentialsBinding(name, namespace, secretName string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"credentialsRef": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
			},
		},
	}
	u.SetLabels(labels)
	u.SetGroupVersionKind(gardener.CredentialsBindingGVK)

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

func NewProviderSpecWithZonesDiscovery(t *testing.T, zonesDiscovery bool) *configuration.ProviderSpec {
	spec := fmt.Sprintf(`
aws:
  zonesDiscovery: %t
  machinesVersions:
    mi.{size}: m7i.{size}
  regionsSupportingMachine:
    ri: {}
`, zonesDiscovery)
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(spec))
	require.NoError(t, err)
	return providerSpec
}

type FakeChannelResolver struct{}

func (f *FakeChannelResolver) GetChannelForPlan(planID string) (string, error) {
	return "fast", nil
}
