package fixture

import (
	"fmt"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

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

func FixInstance(id string) internal.Instance {
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
		Parameters:                  FixProvisioningParameters(id),
		ProviderRegion:              Region,
		Provider:                    pkg.Azure,
		InstanceDetails:             FixInstanceDetails(id),
		CreatedAt:                   time.Now(),
		UpdatedAt:                   time.Now().Add(time.Minute * 5),
		Version:                     0,
	}
}
