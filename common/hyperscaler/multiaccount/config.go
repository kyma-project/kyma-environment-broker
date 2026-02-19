package multiaccount

type MultiAccountConfig struct {
	AllowedGlobalAccounts    []string
	HyperscalerAccountLimits HyperscalerAccountLimits
}

type HyperscalerAccountLimits struct {
	Default int `envconfig:"default=100"`

	AWS       int `envconfig:"optional"`
	GCP       int `envconfig:"optional"`
	Azure     int `envconfig:"optional"`
	OpenStack int `envconfig:"optional"`
	AliCloud  int `envconfig:"optional"`
}
