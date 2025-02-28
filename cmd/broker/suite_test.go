package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	hyperscalerautomock "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/automock"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	kebOrchestration "github.com/kyma-project/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/mock"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	globalAccountLabel   = "account"
	subAccountLabel      = "subaccount"
	runtimeIDAnnotation  = "kcp.provisioner.kyma-project.io/runtime-id"
	defaultKymaVer       = "2.4.0"
	defaultRegion        = "cf-eu10"
	globalAccountID      = "dummy-ga-id"
	dashboardURL         = "http://console.garden-dummy.kyma.io"
	operationID          = "provisioning-op-id"
	deprovisioningOpID   = "deprovisioning-op-id"
	reDeprovisioningOpID = "re-deprovisioning-op-id"
	instanceID           = "instance-id"
	dbSecretKey          = "1234567890123456"

	pollingInterval = 3 * time.Millisecond
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

	orchestrationConfig := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "orchestration-config",
			Namespace: "kcp-system",
			Labels:    map[string]string{},
		},
		Data: map[string]string{
			"maintenancePolicy": `{
	      "rules": [

	      ],
	      "default": {
	        "days": ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
	          "timeBegin": "010000+0000",
	          "timeEnd": "010000+0000"
	      }
	    }`,
		},
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
      channel: stable
      modules:
          - name: btp-operator
            customResourcePolicy: CreateAndDelete
          - name: keda
            channel: fast
`,
		},
	}

	for _, version := range additionalKymaVersions {
		kebCfg.ObjectMeta.Labels[fmt.Sprintf("runtime-version-%s", version)] = "true"
	}

	resources = append(resources, override, scOverride, orchestrationConfig, kebCfg)

	return resources
}

func regularSubscription(ht hyperscaler.Type) string {
	return fmt.Sprintf("regular-%s", ht.GetKey())
}

func sharedSubscription(ht hyperscaler.Type) string {
	return fmt.Sprintf("shared-%s", ht.GetKey())
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
	}

	kimConfigPlans := []string{
		"preview",
		"aws",
		"gcp",
		"azure",
		"trial",
		"free",
		"sap-converged-cloud",
		"azure_lite",
		"build-runtime-aws",
		"build-runtime-gcp",
		"build-runtime-azure",
	}

	return &Config{
		DbInMemory:                         true,
		DisableProcessOperationsInProgress: false,
		DevelopmentMode:                    true,
		DumpProvisionerRequests:            true,
		OperationTimeout:                   2 * time.Minute,
		Provisioner: input.Config{
			ProvisioningTimeout:                     2 * time.Minute,
			DeprovisioningTimeout:                   2 * time.Minute,
			GardenerClusterStepTimeout:              time.Second,
			MachineImage:                            "gardenlinux",
			MachineImageVersion:                     "12345.6",
			MultiZoneCluster:                        true,
			RuntimeResourceStepTimeout:              300 * time.Millisecond,
			ClusterUpdateStepTimeout:                time.Minute,
			CheckRuntimeResourceDeletionStepTimeout: 50 * time.Millisecond,
			DefaultTrialProvider:                    "AWS",
			ControlPlaneFailureTolerance:            "zone",
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
			EnablePlans:                           brokerConfigPlans,
			AllowUpdateExpiredInstanceWithContext: true,
			Binding: broker.BindingConfig{
				Enabled:              true,
				BindablePlans:        []string{"aws", "azure"},
				ExpirationSeconds:    600,
				MaxExpirationSeconds: 7200,
				MinExpirationSeconds: 600,
				MaxBindingsCount:     10,
				CreateBindingTimeout: 15 * time.Second,
			},
			KimConfig: broker.KimConfig{
				Enabled:      true,
				Plans:        kimConfigPlans,
				KimOnlyPlans: kimConfigPlans,
			},
			WorkerHealthCheckInterval:     10 * time.Minute,
			WorkerHealthCheckWarnInterval: 10 * time.Minute,
		},
		Notification: notification.Config{
			Url: "http://host:8080/",
		},
		OrchestrationConfig: kebOrchestration.Config{
			Namespace: "kcp-system",
			Name:      "orchestration-config",
		},
		TrialRegionMappingFilePath:                "testdata/trial-regions.yaml",
		SapConvergedCloudRegionMappingsFilePath:   "testdata/old-sap-converged-cloud-region-mappings.yaml",
		MaxPaginationPage:                         100,
		FreemiumProviders:                         []string{"aws", "azure"},
		FreemiumWhitelistedGlobalAccountsFilePath: "testdata/freemium_whitelist.yaml",
		RegionsSupportingMachineFilePath:          "testdata/regions-supporting-machine.yaml",
		Provisioning:                              process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		Deprovisioning:                            process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		Update:                                    process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		ArchiveEnabled:                            true,
		CleaningEnabled:                           true,
		UpdateRuntimeResourceDelay:                time.Millisecond,
		MetricsV2: metricsv2.Config{
			Enabled:                                         true,
			OperationResultRetentionPeriod:                  time.Hour,
			OperationResultPollingInterval:                  3 * time.Second,
			OperationStatsPollingInterval:                   3 * time.Second,
			OperationResultFinishedOperationRetentionPeriod: time.Hour,
			BindingsStatsPollingInterval:                    3 * time.Second,
		},
	}
}

type LoggingBindingsClient struct {
	requestedLabels []string
}

func (b *LoggingBindingsClient) returnBinding(labelSelector string) (*gardener.SecretBinding, error) {
	if strings.Contains(labelSelector, "shared=true") ||
		strings.Contains(labelSelector, "shared,") || strings.HasSuffix(labelSelector, "shared") {
		return &gardener.SecretBinding{
			Unstructured: unstructured.Unstructured{
				Object: map[string]interface{}{
					"secretRef": map[string]interface{}{
						"name": sharedSubscription(hyperscaler.AWS()), // any hyperscaler
					},
				},
			},
		}, nil
	}

	return &gardener.SecretBinding{
		Unstructured: unstructured.Unstructured{
			Object: map[string]interface{}{
				"secretRef": map[string]interface{}{
					"name": regularSubscription(hyperscaler.AWS()), // any hyperscaler
				},
			},
		},
	}, nil
}

func (b *LoggingBindingsClient) GetSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	b.requestedLabels = append(b.requestedLabels, labelSelector)
	return b.returnBinding(labelSelector)
}

func (b *LoggingBindingsClient) GetSecretBindings(labelSelector string) ([]unstructured.Unstructured, error) {
	b.requestedLabels = append(b.requestedLabels, labelSelector)
	binding, err := b.returnBinding(labelSelector)
	return []unstructured.Unstructured{
		binding.Unstructured,
	}, err
}

func (b *LoggingBindingsClient) GetLeastUsed(secretBindings []unstructured.Unstructured) (*gardener.SecretBinding, error) {
	return &gardener.SecretBinding{secretBindings[0]}, nil
}

func (b *LoggingBindingsClient) UpdateSecretBinding(secretBinding *gardener.SecretBinding) (*unstructured.Unstructured, error) {
	return &secretBinding.Unstructured, nil
}

func (b *LoggingBindingsClient) IsUsedByShoot(secretBinding *gardener.SecretBinding) (bool, error) {
	return false, nil
}

func fixAccountProvider() *hyperscalerautomock.AccountProvider {
	accountProvider := hyperscalerautomock.AccountProvider{}

	accountProvider.On("GardenerSecretName", mock.Anything, mock.Anything, mock.Anything, false).Return(
		func(ht hyperscaler.Type, tn string, euaccess bool, shared bool) string {
			return regularSubscription(ht)
		}, nil)

	accountProvider.On("GardenerSecretName", mock.Anything, mock.Anything, mock.Anything, true).Return(
		func(ht hyperscaler.Type, tn string, euaccess bool, shared bool) string {
			return sharedSubscription(ht)
		}, nil)

	accountProvider.On("GardenerSharedSecretName", hyperscaler.Azure(), mock.Anything).Return(
		func(ht hyperscaler.Type, euaccess bool) string { return sharedSubscription(ht) }, nil)

	accountProvider.On("GardenerSharedSecretName", hyperscaler.AWS(), mock.Anything).Return(
		func(ht hyperscaler.Type, euaccess bool) string { return sharedSubscription(ht) }, nil)

	accountProvider.On("GardenerSharedSecretName", hyperscaler.SapConvergedCloud("eu-de-2"), mock.Anything).Return(
		func(ht hyperscaler.Type, euaccess bool) string { return sharedSubscription(ht) }, nil)

	accountProvider.On("GardenerSharedSecretName", hyperscaler.SapConvergedCloud("eu-de-1"), mock.Anything).Return(
		func(ht hyperscaler.Type, euaccess bool) string { return sharedSubscription(ht) }, nil)

	accountProvider.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.Azure(), mock.Anything, mock.Anything).Return(nil)
	accountProvider.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.AWS(), mock.Anything, mock.Anything).Return(nil)
	return &accountProvider
}
