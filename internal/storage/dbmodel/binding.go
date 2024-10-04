package dbmodel

import (
	"time"
)

type BindingDTO struct {
	ID         string
	InstanceID string

	CreatedAt time.Time

	Kubeconfig        string
	ExpirationSeconds int64
	GenerationMethod  string
	BindingType       string
}
