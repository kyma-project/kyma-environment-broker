package input

import (
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

const (
	trialSuffixLength    = 5
	maxRuntimeNameLength = 36
)

type Config struct {
	// not used
	URL string
	// not used
	ProvisioningTimeout time.Duration `envconfig:"default=6h"`
	// not used
	DeprovisioningTimeout time.Duration `envconfig:"default=5h"`
	// deprecated - is being moved to InfrastructureManager config
	KubernetesVersion string `envconfig:"default=1.16.9"`
	// deprecated - is being moved to InfrastructureManager config
	DefaultGardenerShootPurpose string `envconfig:"default=development"`
	// deprecated - is being moved to InfrastructureManager config
	MachineImage string `envconfig:"optional"`
	// deprecated - is being moved to InfrastructureManager config
	MachineImageVersion string `envconfig:"optional"`
	// not used
	TrialNodesNumber int `envconfig:"optional"`
	// deprecated - is being moved to InfrastructureManager config
	DefaultTrialProvider pkg.CloudProvider `envconfig:"default=Azure"`
	// deprecated - to be removed
	AutoUpdateKubernetesVersion bool `envconfig:"default=false"`
	// deprecated - to be removed
	AutoUpdateMachineImageVersion bool `envconfig:"default=false"`
	// deprecated - is being moved to InfrastructureManager config
	MultiZoneCluster bool `envconfig:"default=false"`
	// deprecated - is being moved to InfrastructureManager config
	ControlPlaneFailureTolerance string `envconfig:"optional"`
	// not used
	GardenerClusterStepTimeout time.Duration `envconfig:"default=3m"`
	// not used
	RuntimeResourceStepTimeout time.Duration `envconfig:"default=8m"`
	// not used
	ClusterUpdateStepTimeout time.Duration `envconfig:"default=2h"`
	// deprecated - is being moved to StepTimeoutsConfig
	CheckRuntimeResourceDeletionStepTimeout time.Duration `envconfig:"default=1h"`
	// deprecated - to be removed
	EnableShootAndSeedSameRegion bool `envconfig:"default=false"`
	// deprecated - is being moved to InfrastructureManager config
	UseMainOIDC bool `envconfig:"default=true"`
	// deprecated - is being moved to InfrastructureManager config
	UseAdditionalOIDC bool `envconfig:"default=false"`
}

type InfrastructureManagerConfig struct {
	KubernetesVersion            string            `envconfig:"default=1.16.9"`
	DefaultGardenerShootPurpose  string            `envconfig:"default=development"`
	MachineImage                 string            `envconfig:"optional"`
	MachineImageVersion          string            `envconfig:"optional"`
	DefaultTrialProvider         pkg.CloudProvider `envconfig:"default=Azure"`
	MultiZoneCluster             bool              `envconfig:"default=false"`
	ControlPlaneFailureTolerance string            `envconfig:"optional"`
	UseMainOIDC                  bool              `envconfig:"default=true"`
	UseAdditionalOIDC            bool              `envconfig:"default=false"`
	UseSmallerMachineTypes       bool              `envconfig:"default=false"`
}
