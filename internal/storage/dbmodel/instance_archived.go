package dbmodel

import (
	"time"

	"github.com/pivotal-cf/brokerapi/v12/domain"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type InstanceArchivedDTO struct {
	InstanceID                  string
	GlobalAccountID             string
	SubaccountID                string
	SubscriptionGlobalAccountID string
	PlanID                      string
	PlanName                    string
	SubaccountRegion            string
	Region                      string
	Provider                    string
	LastRuntimeID               string
	InternalUser                bool
	ShootName                   string

	ProvisioningStartedAt         time.Time
	ProvisioningFinishedAt        time.Time
	ProvisioningState             domain.LastOperationState
	FirstDeprovisioningStartedAt  time.Time
	FirstDeprovisioningFinishedAt time.Time
	LastDeprovisioningFinishedAt  time.Time
}

func NewInstanceArchivedDTO(obj internal.InstanceArchived) InstanceArchivedDTO {
	return InstanceArchivedDTO{
		InstanceID:                    obj.InstanceID,
		GlobalAccountID:               obj.GlobalAccountID,
		SubaccountID:                  obj.SubaccountID,
		SubscriptionGlobalAccountID:   obj.SubscriptionGlobalAccountID,
		PlanID:                        obj.PlanID,
		PlanName:                      obj.PlanName,
		SubaccountRegion:              obj.SubaccountRegion,
		Region:                        obj.Region,
		Provider:                      obj.Provider,
		LastRuntimeID:                 obj.LastRuntimeID,
		InternalUser:                  obj.InternalUser,
		ShootName:                     obj.ShootName,
		ProvisioningStartedAt:         obj.ProvisioningStartedAt,
		ProvisioningFinishedAt:        obj.ProvisioningFinishedAt,
		ProvisioningState:             obj.ProvisioningState,
		FirstDeprovisioningStartedAt:  obj.FirstDeprovisioningStartedAt,
		FirstDeprovisioningFinishedAt: obj.FirstDeprovisioningFinishedAt,
		LastDeprovisioningFinishedAt:  obj.LastDeprovisioningFinishedAt,
	}
}

func NewInstanceArchivedFromDTO(obj InstanceArchivedDTO) internal.InstanceArchived {
	return internal.InstanceArchived{
		InstanceID:                    obj.InstanceID,
		GlobalAccountID:               obj.GlobalAccountID,
		SubaccountID:                  obj.SubaccountID,
		SubscriptionGlobalAccountID:   obj.SubscriptionGlobalAccountID,
		PlanID:                        obj.PlanID,
		PlanName:                      obj.PlanName,
		SubaccountRegion:              obj.SubaccountRegion,
		Region:                        obj.Region,
		Provider:                      obj.Provider,
		LastRuntimeID:                 obj.LastRuntimeID,
		InternalUser:                  obj.InternalUser,
		ShootName:                     obj.ShootName,
		ProvisioningStartedAt:         obj.ProvisioningStartedAt,
		ProvisioningFinishedAt:        obj.ProvisioningFinishedAt,
		ProvisioningState:             obj.ProvisioningState,
		FirstDeprovisioningStartedAt:  obj.FirstDeprovisioningStartedAt,
		FirstDeprovisioningFinishedAt: obj.FirstDeprovisioningFinishedAt,
		LastDeprovisioningFinishedAt:  obj.LastDeprovisioningFinishedAt,
	}
}
