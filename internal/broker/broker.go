package broker

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"gopkg.in/yaml.v2"
)

const (
	KymaServiceID   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	KymaServiceName = "kymaruntime"
	KcpNamespace    = "kcp-system"
)

type KymaEnvironmentBroker struct {
	*ServicesEndpoint
	*ProvisionEndpoint
	*DeprovisionEndpoint
	*UpdateEndpoint
	*GetInstanceEndpoint
	*LastOperationEndpoint
	*BindEndpoint
	*UnbindEndpoint
	*GetBindingEndpoint
	*LastBindingOperationEndpoint
}

// Config represents configuration for broker
type Config struct {
	EnablePlans                             EnablePlans `envconfig:"default=azure"`
	OnlySingleTrialPerGA                    bool        `envconfig:"default=true"`
	URL                                     string
	IncludeAdditionalParamsInSchema         bool          `envconfig:"default=false"`
	ShowTrialExpirationInfo                 bool          `envconfig:"default=false"`
	ShowFreeExpirationInfo                  bool          `envconfig:"default=false"`
	OnlyOneFreePerGA                        bool          `envconfig:"default=false"`
	FreeDocsURL                             string        `envconfig:"default="`
	FreeExpirationPeriod                    time.Duration `envconfig:"default=720h"` // 30 days
	SubaccountsIdsToShowTrialExpirationInfo string        `envconfig:"default="`
	TrialDocsURL                            string        `envconfig:"default="`
	EnableShootAndSeedSameRegion            bool          `envconfig:"default=false"`
	AllowUpdateExpiredInstanceWithContext   bool          `envconfig:"default=false"`
	DefaultRequestRegion                    string        `envconfig:"default=cf-eu10"`
	// OperationTimeout is used to check on a top-level if any operation didn't exceed the time for processing.
	// It is used for provisioning and deprovisioning operations.
	OperationTimeout time.Duration `envconfig:"default=24h"`
	Port             string        `envconfig:"default=8080"`
	StatusPort       string        `envconfig:"default=8071"`
	Host             string        `envconfig:"optional"`

	Binding BindingConfig

	DisableSapConvergedCloud bool `envconfig:"default=false"`

	SubaccountMovementEnabled                bool `envconfig:"default=false"`
	UpdateCustomResourcesLabelsOnAccountMove bool `envconfig:"default=false"`

	WorkerHealthCheckInterval     time.Duration `envconfig:"default=10m"`
	WorkerHealthCheckWarnInterval time.Duration `envconfig:"default=10m"`

	UseAdditionalOIDCSchema bool `envconfig:"default=false"`

	MonitorAdditionalProperties bool   `envconfig:"default=false"`
	AdditionalPropertiesPath    string `envconfig:"default=/additional-properties"`
}

type ServicesConfig map[string]Service

func NewServicesConfigFromFile(path string) (ServicesConfig, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading YAML file with managed components list: %w", err)
	}
	var servicesConfig struct {
		Services ServicesConfig `yaml:"services"`
	}
	err = yaml.Unmarshal(yamlFile, &servicesConfig)
	if err != nil {
		return nil, fmt.Errorf("while unmarshaling YAML file with managed components list: %w", err)
	}
	return servicesConfig.Services, nil
}

func (s ServicesConfig) DefaultPlansConfig() (PlansConfig, error) {
	cfg, ok := s[KymaServiceName]
	if !ok {
		return nil, fmt.Errorf("while getting data about %s plans", KymaServiceName)
	}
	return cfg.Plans, nil
}

type Service struct {
	Description string          `yaml:"description"`
	Metadata    ServiceMetadata `yaml:"metadata"`
	Plans       PlansConfig     `yaml:"plans"`
}

type ServiceMetadata struct {
	DisplayName         string `yaml:"displayName"`
	ImageUrl            string `yaml:"imageUrl"`
	LongDescription     string `yaml:"longDescription"`
	ProviderDisplayName string `yaml:"providerDisplayName"`
	DocumentationUrl    string `yaml:"documentationUrl"`
	SupportUrl          string `yaml:"supportUrl"`
}

type InfrastructureManager struct {
	KubernetesVersion            string            `envconfig:"default=1.16.9"`
	DefaultGardenerShootPurpose  string            `envconfig:"default=development"`
	MachineImage                 string            `envconfig:"optional"`
	MachineImageVersion          string            `envconfig:"optional"`
	DefaultTrialProvider         pkg.CloudProvider `envconfig:"default=Azure"`
	MultiZoneCluster             bool              `envconfig:"default=false"`
	ControlPlaneFailureTolerance string            `envconfig:"optional"`
	UseSmallerMachineTypes       bool              `envconfig:"default=false"`
	EnableIngressFiltering       bool              `envconfig:"default=false"`
	IngressFilteringPlans        EnablePlans       `envconfig:"default=no-plan"`
}

type PlansConfig map[string]PlanData

type PlanData struct {
	Description string       `yaml:"description"`
	Metadata    PlanMetadata `yaml:"metadata"`
}
type PlanMetadata struct {
	DisplayName string `yaml:"displayName"`
}

// EnablePlans defines the plans that should be available for provisioning
type EnablePlans []string

// Unmarshal provides custom parsing of enabled plans.
// Implements envconfig.Unmarshal interface.
func (m *EnablePlans) Unmarshal(in string) error {
	plans := strings.Split(in, ",")
	for _, name := range plans {
		if _, exists := PlanIDsMapping[name]; !exists {
			return fmt.Errorf("unrecognized %v plan name", name)
		}
	}

	*m = plans
	return nil
}

func (m *EnablePlans) ContainsPlanID(PlanID string) bool {
	return m.Contains(PlanNamesMapping[PlanID])
}

func (m *EnablePlans) Contains(name string) bool {
	lowerName := strings.ToLower(name)
	for _, plan := range *m {
		if lowerName == strings.ToLower(plan) {
			return true
		}
	}
	return false
}
