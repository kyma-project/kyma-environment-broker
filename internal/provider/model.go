package provider

// Values contains values which are specific to particular plans (and provisioning parameters)
type Values struct {
	DefaultAutoScalerMax int
	DefaultAutoScalerMin int
	ZonesCount           int
	Zones                []string
	ProviderType         string
	DefaultMachineType   string
	Region               string
	Purpose              string //TODO default per landscape in configuration
	VolumeSizeGb         int
	DiskType             string
	// todo: add other plan specific values
}
