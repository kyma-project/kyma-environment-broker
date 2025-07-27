package ans

import "time"

type Config struct {
	Enabled                bool
	ClientID               string
	ClientSecret           string
	AuthURL                string
	ServiceURL             string
	RateLimitingInterval   time.Duration `envconfig:"default=2s,optional"`
	MaxRequestsPerInterval int           `envconfig:"default=5,optional"`
}
