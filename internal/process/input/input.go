package input

import (
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

type Config struct {
	URL                                     string
	ProvisioningTimeout                     time.Duration     `envconfig:"default=6h"`
	DeprovisioningTimeout                   time.Duration     `envconfig:"default=5h"`
	KubernetesVersion                       string            `envconfig:"default=1.16.9"`
	DefaultGardenerShootPurpose             string            `envconfig:"default=development"`
	MachineImage                            string            `envconfig:"optional"`
	MachineImageVersion                     string            `envconfig:"optional"`
	TrialNodesNumber                        int               `envconfig:"optional"`
	DefaultTrialProvider                    pkg.CloudProvider `envconfig:"default=Azure"`
	AutoUpdateKubernetesVersion             bool              `envconfig:"default=false"`
	AutoUpdateMachineImageVersion           bool              `envconfig:"default=false"`
	MultiZoneCluster                        bool              `envconfig:"default=false"`
	ControlPlaneFailureTolerance            string            `envconfig:"optional"`
	GardenerClusterStepTimeout              time.Duration     `envconfig:"default=3m"`
	RuntimeResourceStepTimeout              time.Duration     `envconfig:"default=8m"`
	ClusterUpdateStepTimeout                time.Duration     `envconfig:"default=2h"`
	CheckRuntimeResourceDeletionStepTimeout time.Duration     `envconfig:"default=1h"`
	EnableShootAndSeedSameRegion            bool              `envconfig:"default=false"`
	UseMainOIDC                             bool              `envconfig:"default=true"`
	UseAdditionalOIDC                       bool              `envconfig:"default=false"`
}
