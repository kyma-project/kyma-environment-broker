package dbmodel

import (
	"database/sql"
	"time"
)

type BindingDTO struct {
	ID         string
	InstanceID string

	CreatedAt time.Time
	ExpiresAt time.Time

	Kubeconfig        string
	ExpirationSeconds int64
	BindingType       string
	Context           sql.NullString
}
