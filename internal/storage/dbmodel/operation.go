package dbmodel

import (
	"database/sql"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

// OperationFilter holds the filters when listing multiple operations
type OperationFilter struct {
	InstanceFilter *InstanceFilter
	Page           int
	PageSize       int
	States         []string
}

type OperationDTO struct {
	ID        string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time

	InstanceID        string
	TargetOperationID string

	Data                   string
	State                  string
	Description            string
	FinishedStages         sql.NullString
	ProvisioningParameters sql.NullString

	Type internal.OperationType

	EncryptionMode string
}

type OperationStatEntry struct {
	Type       string
	State      string
	PlanID     sql.NullString
	InstanceID string
}

type OperationStatEntryV2 struct {
	Count  int
	Type   string
	State  string
	PlanID sql.NullString
}

type InstanceByGlobalAccountIDStatEntry struct {
	GlobalAccountID string
	Total           int
}

type InstanceBySubAccountIDStatEntry struct {
	SubAccountID string
	Total        int
}

type InstanceERSContextStatsEntry struct {
	LicenseType sql.NullString
	Total       int
}
