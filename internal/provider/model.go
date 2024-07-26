package provider

// Values contains values which are specific to particular plans (and provisioning parameters)
type Values struct {
	DefaultAutoScalerMax int
	DefaultAutoScalerMin int
	ZonesCount           int //TODO do we need this as separate value?
	Zones                []string
	ProviderType         string
	DefaultMachineType   string
	Region               string
	Purpose              string //TODO default per landscape in configuration

	// todo: add other plan specific values
}
