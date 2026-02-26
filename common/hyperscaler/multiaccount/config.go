package multiaccount

type MultiAccountConfig struct {
	AllowedGlobalAccounts []string
	Limits                HyperscalerAccountLimits
}

type HyperscalerAccountLimits struct {
	Default int `envconfig:"default=999999"`

	AWS       int `envconfig:"optional"`
	GCP       int `envconfig:"optional"`
	Azure     int `envconfig:"optional"`
	OpenStack int `envconfig:"optional"`
	AliCloud  int `envconfig:"optional"`
}
