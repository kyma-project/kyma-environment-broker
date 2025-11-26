package dbmodel

import (
	"database/sql"
	"time"
)

type InstanceState string

const (
	InstanceSucceeded        InstanceState = "succeeded"
	InstanceFailed           InstanceState = "failed"
	InstanceError            InstanceState = "error"
	InstanceProvisioning     InstanceState = "provisioning"
	InstanceDeprovisioning   InstanceState = "deprovisioning"
	InstanceUpgrading        InstanceState = "upgrading"
	InstanceUpdating         InstanceState = "updating"
	InstanceDeprovisioned    InstanceState = "deprovisioned"
	InstanceNotDeprovisioned InstanceState = "notDeprovisioned"
)

// InstanceFilter holds the filters when querying Instances
type InstanceFilter struct {
	PageSize                     int
	Page                         int
	GlobalAccountIDs             []string
	SubscriptionGlobalAccountIDs []string
	SubAccountIDs                []string
	InstanceIDs                  []string
	RuntimeIDs                   []string
	Regions                      []string
	PlanIDs                      []string
	Plans                        []string
	Shoots                       []string
	States                       []InstanceState
	Expired                      *bool
	DeletionAttempted            *bool
	BindingExists                *bool
	Suspended                    *bool
}

type InstanceDTO struct {
	InstanceID                  string
	RuntimeID                   string
	GlobalAccountID             string
	SubscriptionGlobalAccountID string
	SubAccountID                string
	ServiceID                   string
	ServiceName                 string
	ServicePlanID               string
	ServicePlanName             string
	SubscriptionSecretName      string

	DashboardURL           string
	ProvisioningParameters string
	ProviderRegion         string
	Provider               string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	ExpiredAt *time.Time

	Version int

	EncryptionMode string
}

type InstanceWithOperationDTO struct {
	InstanceDTO

	Type               sql.NullString
	State              sql.NullString
	OperationCreatedAt sql.NullTime
	Data               sql.NullString
	Description        sql.NullString
}

type InstanceWithExtendedOperationDTO struct {
	InstanceDTO
	OperationDTO
}

type InstanceWithSubaccountStateDTO struct {
	InstanceDTO
	OperationDTO
	BetaEnabled       *string
	UsedForProduction *string
}
