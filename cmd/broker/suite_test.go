package main

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/metrics"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	globalAccountLabel    = "account"
	subAccountLabel       = "subaccount"
	runtimeIDAnnotation   = "kcp.provisioner.kyma-project.io/runtime-id"
	defaultKymaVer        = "2.4.0"
	operationID           = "provisioning-op-id"
	instanceID            = "instance-id"
	dbSecretKey           = "1234567890123456"
	gardenerKymaNamespace = "kyma"
)

var (
	shootGVK = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "Shoot"}
)

type RuntimeOptions struct {
	GlobalAccountID  string
	SubAccountID     string
	PlatformProvider pkg.CloudProvider
	PlatformRegion   string
	Region           string
	PlanID           string
	Provider         pkg.CloudProvider
	OIDC             *pkg.OIDCConfigDTO
	UserID           string
	RuntimeAdmins    []string
}

func (o *RuntimeOptions) ProvideGlobalAccountID() string {
	if o.GlobalAccountID != "" {
		return o.GlobalAccountID
	} else {
		return uuid.New().String()
	}
}

func (o *RuntimeOptions) ProvideSubAccountID() string {
	if o.SubAccountID != "" {
		return o.SubAccountID
	} else {
		return uuid.New().String()
	}
}

func (o *RuntimeOptions) ProvidePlatformRegion() string {
	if o.PlatformProvider != "" {
		return o.PlatformRegion
	} else {
		return "cf-eu10"
	}
}

func (o *RuntimeOptions) ProvideRegion() *string {
	if o.Region != "" {
		return &o.Region
	} else {
		r := "westeurope"
		return &r
	}
}

func (o *RuntimeOptions) ProvidePlanID() string {
	if o.PlanID == "" {
		return broker.AzurePlanID
	} else {
		return o.PlanID
	}
}

func (o *RuntimeOptions) ProvideOIDC() *pkg.OIDCConfigDTO {
	if o.OIDC != nil {
		return o.OIDC
	} else {
		return nil
	}
}

func (o *RuntimeOptions) ProvideUserID() string {
	return o.UserID
}

func (o *RuntimeOptions) ProvideRuntimeAdmins() []string {
	if o.RuntimeAdmins != nil {
		return o.RuntimeAdmins
	} else {
		return nil
	}
}

func fixK8sResources(defaultKymaVersion string, additionalKymaVersions []string) []runtime.Object {
	var resources []runtime.Object
	override := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "overrides",
			Namespace: "kcp-system",
			Labels: map[string]string{
				fmt.Sprintf("overrides-version-%s", defaultKymaVersion): "true",
				"overrides-plan-azure":               "true",
				"overrides-plan-trial":               "true",
				"overrides-plan-aws":                 "true",
				"overrides-plan-free":                "true",
				"overrides-plan-gcp":                 "true",
				"overrides-plan-own_cluster":         "true",
				"overrides-plan-sap-converged-cloud": "true",
				"overrides-version-2.0.0-rc4":        "true",
				"overrides-version-2.0.0":            "true",
			},
		},
		Data: map[string]string{
			"foo":                            "bar",
			"global.booleanOverride.enabled": "false",
		},
	}
	scOverride := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "service-catalog2-overrides",
			Namespace: "kcp-system",
			Labels: map[string]string{
				fmt.Sprintf("overrides-version-%s", defaultKymaVersion): "true",
				"overrides-plan-azure":        "true",
				"overrides-plan-trial":        "true",
				"overrides-plan-aws":          "true",
				"overrides-plan-free":         "true",
				"overrides-plan-gcp":          "true",
				"overrides-version-2.0.0-rc4": "true",
				"overrides-version-2.0.0":     "true",
				"component":                   "service-catalog2",
			},
		},
		Data: map[string]string{
			"setting-one": "1234",
		},
	}

	for _, version := range additionalKymaVersions {
		override.ObjectMeta.Labels[fmt.Sprintf("overrides-version-%s", version)] = "true"
		scOverride.ObjectMeta.Labels[fmt.Sprintf("overrides-version-%s", version)] = "true"
	}

	kebCfg := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "keb-runtime-config",
			Namespace: "kcp-system",
			Labels: map[string]string{
				"keb-config": "true",
			},
		},
		Data: map[string]string{
			"default": `
kyma-template: |-
  apiVersion: operator.kyma-project.io/v1beta2
  kind: Kyma
  metadata:
      name: my-kyma
      namespace: kyma-system
  spec:
      sync:
          strategy: secret
      channel: fast
      modules:
          - name: btp-operator
            customResourcePolicy: CreateAndDelete
          - name: keda
            channel: fast
`,
		},
	}

	providerCfg := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "gardener-seeds-cache",
			Namespace: "kcp-system",
		},
		Data: map[string]string{
			"aws": `
seedRegions:
- eu-central-1`,
			"azure": `
seedRegions:
- westeurope`,
			"gcp": `
seedRegions:
- europe-west1`,
			"openstack": `
seedRegions:
- eu-de-1`,
			"alicloud": `
seedRegions:
- eu-central-1`,
		},
	}

	for _, version := range additionalKymaVersions {
		kebCfg.ObjectMeta.Labels[fmt.Sprintf("runtime-version-%s", version)] = "true"
	}

	resources = append(resources, override, scOverride, kebCfg, providerCfg)

	return resources
}

func fixConfig() *Config {
	brokerConfigPlans := []string{
		"azure",
		"trial",
		"aws",
		"own_cluster",
		"preview",
		"sap-converged-cloud",
		"gcp",
		"free",
		"build-runtime-aws",
		"build-runtime-gcp",
		"build-runtime-azure",
		"alicloud",
	}

	return &Config{
		DbInMemory:                         true,
		DisableProcessOperationsInProgress: false,
		DevelopmentMode:                    true,
		InfrastructureManager: broker.InfrastructureManager{
			MachineImage:                 "gardenlinux",
			MachineImageVersion:          "12345.6",
			MultiZoneCluster:             true,
			DefaultTrialProvider:         "AWS",
			ControlPlaneFailureTolerance: "zone",
			IngressFilteringPlans:        []string{"aws", "azure", "gcp"},
		},
		StepTimeouts: StepTimeoutsConfig{
			CheckRuntimeResourceUpdate:   180 * time.Second,
			CheckRuntimeResourceCreate:   60 * time.Second,
			CheckRuntimeResourceDeletion: 50 * time.Millisecond,
		},
		Database: storage.Config{
			SecretKey: dbSecretKey,
		},
		Gardener: gardener.Config{
			Project:     "kyma",
			ShootDomain: "kyma.sap.com",
		},
		UpdateProcessingEnabled: true,
		Broker: broker.Config{
			EnablePlans:      brokerConfigPlans,
			OperationTimeout: 2 * time.Minute,
			Binding: broker.BindingConfig{
				Enabled:              true,
				BindablePlans:        []string{"aws", "azure"},
				ExpirationSeconds:    600,
				MaxExpirationSeconds: 7200,
				MinExpirationSeconds: 600,
				MaxBindingsCount:     10,
				CreateBindingTimeout: 15 * time.Second,
			},
			GardenerSeedsCacheConfigMapName: "gardener-seeds-cache",
			EnablePlanUpgrades:              true,
			DualStackDocsURL:                "https://placeholder.com",
		},
		TrialRegionMappingFilePath:                "testdata/trial-regions.yaml",
		MaxPaginationPage:                         100,
		FreemiumWhitelistedGlobalAccountsFilePath: "testdata/freemium_whitelist.yaml",
		Provisioning:                              process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		Deprovisioning:                            process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		Update:                                    process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		UpdateRuntimeResourceDelay:                time.Millisecond,
		Metrics: metrics.Config{
			Enabled:                                         true,
			OperationResultRetentionPeriod:                  time.Hour,
			OperationResultPollingInterval:                  3 * time.Second,
			OperationStatsPollingInterval:                   3 * time.Second,
			OperationResultFinishedOperationRetentionPeriod: time.Hour,
			BindingsStatsPollingInterval:                    3 * time.Second,
		},
		ProvidersConfigurationFilePath:               "testdata/providers.yaml",
		PlansConfigurationFilePath:                   "testdata/plans.yaml",
		RuntimeConfigurationConfigMapName:            "keb-runtime-config",
		QuotaWhitelistedSubaccountsFilePath:          "testdata/quota_whitelist.yaml",
		SubscriptionGardenerResource:                 "secretbinding",
		MachinesAvailabilityEndpoint:                 true,
		BtpRegionsMigrationSapConvergedCloudFilePath: "testdata/btp-regions-migration-sap-converged-cloud.yaml",
	}
}
