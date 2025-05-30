package fixture

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

const (
	ServiceId                   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	ServiceName                 = "kymaruntime"
	PlanId                      = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	TrialPlan                   = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	PlanName                    = "azure"
	GlobalAccountId             = "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4"
	SubscriptionGlobalAccountID = ""
	Region                      = "westeurope"
	ServiceManagerUsername      = "u"
	ServiceManagerPassword      = "p"
	ServiceManagerURL           = "https://service-manager.local"
	InstanceDashboardURL        = "https://dashboard.local"
	XSUAADataXSAppName          = "XSApp"
	MonitoringUsername          = "username"
	MonitoringPassword          = "password"
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
		VolumeSizeGb:     ptr.Integer(50),
		MachineType:      ptr.String("Standard_D8_v3"),
		Region:           ptr.String(Region),
		Purpose:          ptr.String("Purpose"),
		LicenceType:      ptr.String("LicenceType"),
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
