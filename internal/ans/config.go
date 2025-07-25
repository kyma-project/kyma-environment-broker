package ans

import "time"

type (
	EndpointConfig struct {
		ClientID               string
		ClientSecret           string
		AuthURL                string
		ServiceURL             string
		RateLimitingInterval   time.Duration `envconfig:"default=2s,optional"`
		MaxRequestsPerInterval int           `envconfig:"default=5,optional"`
	}
	Config struct {
		Enabled       bool
		Events        EndpointConfig
		Notifications EndpointConfig
	}
)
