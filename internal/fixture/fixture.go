package fixture

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/pivotal-cf/brokerapi/v12/domain"
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
	SubscriptionGlobalAccountID    = ""
	Region                         = "westeurope"
	ServiceManagerUsername         = "u"
	ServiceManagerPassword         = "p"
	ServiceManagerURL              = "https://service-manager.local"
	InstanceDashboardURL           = "https://dashboard.local"
	XSUAADataXSAppName             = "XSApp"
	MonitoringUsername             = "username"
	MonitoringPassword             = "password"
	AWSTenantName                  = "aws-tenant-1"
	AzureTenantName                = "azure-tenant-2"
	AWSSecretName                  = "aws-secret"
	AWSEUAccessClaimedSecretName   = "aws-euaccess-tenant-1"
	AzureEUAccessClaimedSecretName = "azure-euaccess-tenant-2"
	AzureUnclaimedSecretName       = "azure-unclaimed"
	GCPEUAccessSharedSecretName    = "gcp-euaccess-shared"
	AWSMostUsedSharedSecretName    = "aws-most-used-shared"
	AWSLeastUsedSharedSecretName   = "aws-least-used-shared"
)

func FixServiceManagerEntryDTO() *internal.ServiceManagerEntryDTO {
	return &internal.ServiceManagerEntryDTO{
		Credentials: internal.ServiceManagerCredentials{
			BasicAuth: internal.ServiceManagerBasicAuth{
				Username: ServiceManagerUsername,
				Password: ServiceManagerPassword,
			},
		},
		URL: ServiceManagerURL,
	}
}

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

func FixProvisioningParametersWithDTO(id string, planID string, params pkg.ProvisioningParametersDTO) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:         planID,
		ServiceID:      ServiceId,
		ErsContext:     FixERSContext(id),
		Parameters:     params,
		PlatformRegion: Region,
	}
}

func FixProvisioningParameters(id string) internal.ProvisioningParameters {
	trialCloudProvider := pkg.Azure

	provisioningParametersDTO := pkg.ProvisioningParametersDTO{
		Name:             "cluster-test",
		MachineType:      ptr.String("Standard_D8_v3"),
		Region:           ptr.String(Region),
		Purpose:          ptr.String("Purpose"),
		Zones:            []string{"1"},
		IngressFiltering: ptr.Bool(true),
		AutoScalerParameters: pkg.AutoScalerParameters{
			AutoScalerMin:  ptr.Integer(3),
			AutoScalerMax:  ptr.Integer(10),
			MaxSurge:       ptr.Integer(4),
			MaxUnavailable: ptr.Integer(1),
		},
		Provider: &trialCloudProvider,
	}

	return internal.ProvisioningParameters{
		PlanID:         PlanId,
		ServiceID:      ServiceId,
		ErsContext:     FixERSContext(id),
		Parameters:     provisioningParametersDTO,
		PlatformRegion: Region,
	}
}

func FixInstanceDetails(id string) internal.InstanceDetails {
	var (
		runtimeId    = fmt.Sprintf("runtime-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
		shootName    = fmt.Sprintf("Shoot-%s", id)
		shootDomain  = fmt.Sprintf("shoot-%s.domain.com", id)
	)

	monitoringData := internal.MonitoringData{
		Username: MonitoringUsername,
		Password: MonitoringPassword,
	}

	return internal.InstanceDetails{
		EventHub:              internal.EventHub{Deleted: false},
		SubAccountID:          subAccountId,
		RuntimeID:             runtimeId,
		ShootName:             shootName,
		ShootDomain:           shootDomain,
		ShootDNSProviders:     FixDNSProvidersConfig(),
		Monitoring:            monitoringData,
		KymaResourceNamespace: "kyma-system",
		KymaResourceName:      runtimeId,
	}
}

func FixInstanceWithProvisioningParameters(id string, params internal.ProvisioningParameters) internal.Instance {
	var (
		runtimeId    = fmt.Sprintf("runtime-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
	)

	return internal.Instance{
		InstanceID:                  id,
		RuntimeID:                   runtimeId,
		GlobalAccountID:             GlobalAccountId,
		SubscriptionGlobalAccountID: SubscriptionGlobalAccountID,
		SubAccountID:                subAccountId,
		ServiceID:                   ServiceId,
		ServiceName:                 ServiceName,
		ServicePlanID:               PlanId,
		ServicePlanName:             PlanName,
		SubscriptionSecretName:      SubscriptionSecretName,
		DashboardURL:                InstanceDashboardURL,
		Parameters:                  params,
		ProviderRegion:              Region,
		Provider:                    pkg.Azure,
		InstanceDetails:             FixInstanceDetails(id),
		CreatedAt:                   time.Now(),
		UpdatedAt:                   time.Now().Add(time.Minute * 5),
		Version:                     0,
	}
}

func FixInstance(id string) internal.Instance {
	return FixInstanceWithProvisioningParameters(id, FixProvisioningParameters(id))
}

func FixOperationWithProvisioningParameters(id, instanceId string, opType internal.OperationType, params internal.ProvisioningParameters) internal.Operation {
	var (
		description = fmt.Sprintf("Description for operation %s", id)
	)

	return internal.Operation{
		InstanceDetails:        FixInstanceDetails(instanceId),
		ID:                     id,
		Type:                   opType,
		Version:                0,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now().Add(time.Hour * 48),
		InstanceID:             instanceId,
		ProvisionerOperationID: "",
		State:                  domain.Succeeded,
		Description:            description,
		ProvisioningParameters: params,
		FinishedStages:         []string{"prepare", "check_provisioning"},
	}
}

func FixOperation(id, instanceId string, opType internal.OperationType) internal.Operation {
	return FixOperationWithProvisioningParameters(id, instanceId, opType, FixProvisioningParameters(id))
}

func FixProvisioningOperation(operationId, instanceId string) internal.Operation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeProvision)
	o.DashboardURL = "https://console.kyma.org"
	return o
}

func FixProvisioningOperationWithProvisioningParameters(operationId, instanceId string, provisioningParameters internal.ProvisioningParameters) internal.Operation {
	o := FixOperationWithProvisioningParameters(operationId, instanceId, internal.OperationTypeProvision, provisioningParameters)
	o.DashboardURL = "https://console.kyma.org"
	return o
}

func FixUpdatingOperation(operationId, instanceId string) internal.UpdatingOperation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeUpdate)
	o.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			List: []pkg.OIDCConfigDTO{
				{
					ClientID:       "clinet-id-oidc",
					GroupsClaim:    "groups",
					IssuerURL:      "issuer-url",
					SigningAlgs:    []string{"signingAlgs"},
					UsernameClaim:  "sub",
					UsernamePrefix: "",
				},
			},
		},
	}
	return internal.UpdatingOperation{
		Operation: o,
	}
}

func FixUpdatingOperationWithOIDCObject(operationId, instanceId string) internal.UpdatingOperation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeUpdate)
	o.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			OIDCConfigDTO: &pkg.OIDCConfigDTO{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				GroupsPrefix:   "-",
				IssuerURL:      "issuer-url",
				SigningAlgs:    []string{"signingAlgs"},
				UsernameClaim:  "sub",
				UsernamePrefix: "",
				RequiredClaims: []string{"claim1=value1", "claim2=value2"},
			},
		},
	}
	return internal.UpdatingOperation{
		Operation: o,
	}
}

func FixProvisioningOperationWithProvider(operationId, instanceId string, provider pkg.CloudProvider) internal.Operation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeProvision)
	o.DashboardURL = "https://console.kyma.org"
	return o
}

func FixDeprovisioningOperation(operationId, instanceId string) internal.DeprovisioningOperation {
	return internal.DeprovisioningOperation{
		Operation: FixDeprovisioningOperationAsOperation(operationId, instanceId),
	}
}

func FixDeprovisioningOperationAsOperation(operationId, instanceId string) internal.Operation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeDeprovision)
	o.Temporary = false
	return o
}

func FixSuspensionOperationAsOperation(operationId, instanceId string) internal.Operation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeDeprovision)
	o.Temporary = true
	o.ProvisioningParameters.PlanID = TrialPlan
	return o
}

func FixUpgradeClusterOperation(operationId, instanceId string) internal.UpgradeClusterOperation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeUpgradeCluster)
	o.RuntimeOperation = FixRuntimeOperation()
	return internal.UpgradeClusterOperation{
		Operation: o,
	}
}

func FixRuntimeOperation() internal.RuntimeOperation {
	return internal.RuntimeOperation{
		GlobalAccountID: GlobalAccountId,
		Region:          Region,
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

func FixBinding(id string) internal.Binding {
	var instanceID = fmt.Sprintf("instance-%s", id)

	return FixBindingWithInstanceID(id, instanceID)
}

func FixBindingWithInstanceID(bindingID string, instanceID string) internal.Binding {
	return internal.Binding{
		ID:         bindingID,
		InstanceID: instanceID,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now().Add(time.Minute * 5),
		ExpiresAt: time.Now().Add(time.Minute * 10),

		Kubeconfig:        "kubeconfig",
		ExpirationSeconds: 600,
		CreatedBy:         "john.smith@email.com",
	}
}

func FixExpiredBindingWithInstanceID(bindingID string, instanceID string, offset time.Duration) internal.Binding {
	return internal.Binding{
		ID:         bindingID,
		InstanceID: instanceID,

		CreatedAt: time.Now().Add(-offset),
		UpdatedAt: time.Now().Add(time.Minute*5 - offset),
		ExpiresAt: time.Now().Add(time.Minute*10 - offset),

		Kubeconfig:        "kubeconfig",
		ExpirationSeconds: 600,
		CreatedBy:         "john.smith@email.com",
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

func NewFakeAWSClientFactory(zones map[string][]string, error error) *FakeAWSClientFactory {
	fakeClient := &fakeAWSClient{
		zones: zones,
		err:   error,
	}
	return &FakeAWSClientFactory{client: fakeClient}
}

type FakeAWSClientFactory struct {
	client aws.Client
}

func (f *FakeAWSClientFactory) New(ctx context.Context, accessKeyID, secretAccessKey, region string) (aws.Client, error) {
	return f.client, nil
}

type fakeAWSClient struct {
	zones map[string][]string
	err   error
}

func (f *fakeAWSClient) AvailableZones(ctx context.Context, machineType string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.zones[machineType], nil
}

func (f *fakeAWSClient) AvailableZonesCount(ctx context.Context, machineType string) (int, error) {
	zones, err := f.AvailableZones(ctx, machineType)
	if err != nil {
		return 0, err
	}
	return len(zones), nil
}

func CreateGardenerClient() *gardener.Client {
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
	sb1 := createSecretBinding(AWSEUAccessClaimedSecretName, namespace, secretName1, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	sb2 := createSecretBinding(AzureEUAccessClaimedSecretName, namespace, secretName2, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
		gardener.EUAccessLabelKey:        "true",
		gardener.TenantNameLabelKey:      AzureTenantName,
	})
	sb3 := createSecretBinding(AzureUnclaimedSecretName, namespace, secretName3, map[string]string{
		gardener.HyperscalerTypeLabelKey: "azure",
	})
	sb4 := createSecretBinding(GCPEUAccessSharedSecretName, namespace, secretName4, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
		gardener.EUAccessLabelKey:        "true",
		gardener.SharedLabelKey:          "true",
	})
	sb5 := createSecretBinding(AWSMostUsedSharedSecretName, namespace, secretName5, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb6 := createSecretBinding(AWSLeastUsedSharedSecretName, namespace, secretName6, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.SharedLabelKey:          "true",
	})
	sb7 := createSecretBinding("", namespace, secretName7, map[string]string{
		gardener.HyperscalerTypeLabelKey: "gcp",
	})
	sb8 := createSecretBinding(AWSSecretName, namespace, secretName8, map[string]string{
		gardener.HyperscalerTypeLabelKey: "aws",
		gardener.TenantNameLabelKey:      AWSTenantName,
	})
	shoot1 := createShoot("shoot-1", namespace, AWSMostUsedSharedSecretName)
	shoot2 := createShoot("shoot-2", namespace, AWSMostUsedSharedSecretName)
	shoot3 := createShoot("shoot-3", namespace, AWSLeastUsedSharedSecretName)

	fakeGardenerClient := gardener.NewDynamicFakeClient(s1, s2, s3, s4, s5, s6, s7, s8, sb1, sb2, sb3, sb4, sb5, sb6, sb7, sb8, shoot1, shoot2, shoot3)

	return gardener.NewClient(fakeGardenerClient, namespace)
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
`, zonesDiscovery)
	providerSpec, err := configuration.NewProviderSpec(strings.NewReader(spec))
	require.NoError(t, err)
	return providerSpec
}

type FakeChannelResolver struct{}

func (f *FakeChannelResolver) GetChannelForPlan(planID string) (string, error) {
	return "fast", nil
}
