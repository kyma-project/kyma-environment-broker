package dbmodel

import (
	"time"
)

type BindingDTO struct {
	ID         string
	InstanceID string

	CreatedAt time.Time
	ExpiresAt time.Time

	Kubeconfig        string
	ExpirationSeconds int64
	CreatedBy         string
}

type BindingStatsDTO struct {
	SecondsSinceEarliestExpiration *float64 `db:"seconds_since_earliest_expiration"`
}
