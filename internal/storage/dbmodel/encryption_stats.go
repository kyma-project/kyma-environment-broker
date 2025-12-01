package dbmodel

import "database/sql"

type EncryptionModeStatsDTO struct {
	EncryptionMode sql.NullString
	Total          int
}
