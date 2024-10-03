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
	Purpose              string
	VolumeSizeGb         int
	DiskType             string
}
