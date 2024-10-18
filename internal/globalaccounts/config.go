package globalaccounts

import (
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type Config struct {
	Database     storage.Config
	DryRun       bool   `envconfig:"default=true"`
	ClientID     string `envconfig:"default=account-service-id"`
	ClientSecret string `envconfig:"default=account-service-secret"`
	AuthURL      string `envconfig:"default=url"`
	ServiceURL   string `envconfig:"default=url"`
}
