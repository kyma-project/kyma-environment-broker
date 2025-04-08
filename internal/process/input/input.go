package input

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

const (
	trialSuffixLength    = 5
	maxRuntimeNameLength = 36
)

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
