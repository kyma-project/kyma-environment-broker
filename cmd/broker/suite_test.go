package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	hyperscalerautomock "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	kebOrchestration "github.com/kyma-project/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	kebRuntime "github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

type OrchestrationSuite struct {
	gardenerNamespace string
	provisionerClient *provisioner.FakeClient
	clusterQueue      *process.Queue
	storage           storage.BrokerStorage
	gardenerClient    dynamic.Interface

	t *testing.T
}

func (s *OrchestrationSuite) TearDown() {
	if r := recover(); r != nil {
		err := cleanupContainer()
		assert.NoError(s.t, err)
		panic(r)
	}
}

func NewOrchestrationSuite(t *testing.T, additionalKymaVersions []string) *OrchestrationSuite {
	defer func() {
		if r := recover(); r != nil {
			err := cleanupContainer()
			assert.NoError(t, err)
			panic(r)
		}
	}()
	logs := logrus.New()
	logs.Formatter.(*logrus.TextFormatter).TimestampFormat = "15:04:05.000"

	var cfg Config
	cfg.OrchestrationConfig = kebOrchestration.Config{
		KubernetesVersion: "",
	}
	cfg.Notification = notification.Config{
		Url: "",
	}

	oidcDefaults := fixture.FixOIDCConfigDTO()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	storageCleanup, db, err := GetStorageForE2ETests()
	assert.NoError(t, err)
	t.Cleanup(func() {
		defer cancel()
		if storageCleanup != nil {
			err := storageCleanup()
			assert.NoError(t, err)
		}
	})
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))

	kymaVer := "2.4.0"
	cli := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(fixK8sResources(kymaVer, additionalKymaVersions)...).Build()
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logrus.New(), "keb-runtime-config"),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	inputFactory, err := input.NewInputBuilderFactory(configProvider, input.Config{
		MachineImageVersion:         "coreos",
		KubernetesVersion:           "1.18",
		MachineImage:                "253",
		ProvisioningTimeout:         time.Minute,
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
	}, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders, oidcDefaults, cfg.Broker.UseSmallerMachineTypes)
	require.NoError(t, err)

	gardenerClient := gardener.NewDynamicFakeClient()
	provisionerClient := provisioner.NewFakeClient()
	const gardenerProject = "testing"
	gardenerNamespace := fmt.Sprintf("garden-%s", gardenerProject)

	eventBroker := event.NewPubSub(logs)

	runtimeLister := kebOrchestration.NewRuntimeLister(db.Instances(), db.Operations(), kebRuntime.NewConverter(defaultRegion), logs)
	runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient, gardenerNamespace, runtimeLister, logs)

	notificationFakeClient := notification.NewFakeClient()
	notificationBundleBuilder := notification.NewBundleBuilder(notificationFakeClient, cfg.Notification)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory, &upgrade_cluster.TimeSchedule{
		Retry:                 2 * time.Millisecond,
		StatusCheck:           20 * time.Millisecond,
		UpgradeClusterTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeResolver, notificationBundleBuilder, logs, cli, cfg, 1000)

	clusterQueue.SpeedUp(1000)

	return &OrchestrationSuite{
		gardenerNamespace: gardenerNamespace,
		provisionerClient: provisionerClient,
		clusterQueue:      clusterQueue,
		storage:           db,
		gardenerClient:    gardenerClient,

		t: t,
	}
}

type RuntimeOptions struct {
	GlobalAccountID  string
	SubAccountID     string
	PlatformProvider internal.CloudProvider
	PlatformRegion   string
	Region           string
	PlanID           string
	Provider         internal.CloudProvider
	OIDC             *internal.OIDCConfigDTO
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

func (o *RuntimeOptions) ProvideOIDC() *internal.OIDCConfigDTO {
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

func (s *OrchestrationSuite) CreateProvisionedRuntime(options RuntimeOptions) string {
	runtimeID := uuid.New().String()
	shootName := fmt.Sprintf("shoot%s", runtimeID)
	planID := options.ProvidePlanID()
	planName := broker.AzurePlanName
	globalAccountID := options.ProvideGlobalAccountID()
	subAccountID := options.ProvideSubAccountID()
	instanceID := uuid.New().String()
	runtimeStateID := uuid.New().String()
	oidcConfig := fixture.FixOIDCConfigDTO()
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: planID,
		ErsContext: internal.ERSContext{
			SubAccountID:    subAccountID,
			GlobalAccountID: globalAccountID,
		},
		PlatformRegion: options.ProvidePlatformRegion(),
		Parameters: internal.ProvisioningParametersDTO{
			Region: options.ProvideRegion(),
			OIDC:   &oidcConfig,
		},
	}

	instance := internal.Instance{
		RuntimeID:       runtimeID,
		ServicePlanID:   planID,
		ServicePlanName: planName,
		InstanceID:      instanceID,
		GlobalAccountID: globalAccountID,
		SubAccountID:    subAccountID,
		Parameters:      provisioningParameters,
		ProviderRegion:  options.ProvidePlatformRegion(),
		InstanceDetails: internal.InstanceDetails{
			RuntimeID:   runtimeID,
			ShootName:   shootName,
			ShootDomain: "fake.domain",
		},
	}

	provisioningOperation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			State:                  domain.Succeeded,
			ID:                     uuid.New().String(),
			InstanceID:             instanceID,
			ProvisioningParameters: provisioningParameters,
			InstanceDetails: internal.InstanceDetails{
				RuntimeID:   instance.RuntimeID,
				ShootName:   shootName,
				ShootDomain: "fake.domain",
			},
			Type: internal.OperationTypeProvision,
		},
	}
	runtimeState := fixture.FixRuntimeState(runtimeStateID, runtimeID, provisioningOperation.ID)
	runtimeState.ClusterConfig.OidcConfig = &gqlschema.OIDCConfigInput{
		ClientID:       oidcConfig.ClientID,
		GroupsClaim:    oidcConfig.GroupsClaim,
		IssuerURL:      oidcConfig.IssuerURL,
		SigningAlgs:    oidcConfig.SigningAlgs,
		UsernameClaim:  oidcConfig.UsernameClaim,
		UsernamePrefix: oidcConfig.UsernamePrefix,
	}
	shoot := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      shootName,
			"namespace": s.gardenerNamespace,
			"labels": map[string]interface{}{
				globalAccountLabel: globalAccountID,
				subAccountLabel:    subAccountID,
			},
			"annotations": map[string]interface{}{
				runtimeIDAnnotation: runtimeID,
			},
		},
		"spec": map[string]interface{}{
			"region": options.ProvidePlatformRegion(),
			"maintenance": map[string]interface{}{
				"timeWindow": map[string]interface{}{
					"begin": "030000+0000",
					"end":   "040000+0000",
				},
			},
		},
	}}
	shoot.SetGroupVersionKind(shootGVK)

	require.NoError(s.t, s.storage.Instances().Insert(instance))
	require.NoError(s.t, s.storage.Operations().InsertOperation(provisioningOperation.Operation))
	require.NoError(s.t, s.storage.RuntimeStates().Insert(runtimeState))
	_, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(s.gardenerNamespace).Create(context.Background(), shoot, v1.CreateOptions{})
	require.NoError(s.t, err)

	provisioningOperation.InputCreator = fixture.FixInputCreator(internal.Azure)
	_, err = s.provisionerClient.Provision(provisioningOperation)
	require.NoError(s.t, err)

	return runtimeID
}

func (s *OrchestrationSuite) createOrchestration(oType orchestration.Type, queue *process.Queue, params orchestration.Parameters) string {
	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		Type:            oType,
		State:           orchestration.Pending,
		Description:     "started processing of Kyma upgrade",
		Parameters:      params,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(s.t, s.storage.Orchestrations().Insert(o))

	queue.Add(o.OrchestrationID)
	return o.OrchestrationID
}

func (s *OrchestrationSuite) CreateUpgradeClusterOrchestration(params orchestration.Parameters) string {
	return s.createOrchestration(orchestration.UpgradeClusterOrchestration, s.clusterQueue, params)
}

func (s *OrchestrationSuite) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
	err := wait.Poll(time.Millisecond*100, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindInProgressOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, gqlschema.OperationStateSucceeded)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *OrchestrationSuite) FinishUpgradeShootOperationByProvisioner(runtimeID string) {
	s.finishOperationByProvisioner(gqlschema.OperationTypeUpgradeShoot, runtimeID)
}

func (s *OrchestrationSuite) WaitForOrchestrationState(orchestrationID string, state string) {
	var orchestration *internal.Orchestration
	err := wait.PollImmediate(100*time.Millisecond, 2*time.Second, func() (done bool, err error) {
		orchestration, _ = s.storage.Orchestrations().GetByID(orchestrationID)
		return orchestration.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the orchestration expected state %s. The existing orchestration %+v", state, orchestration)
}

func (s *OrchestrationSuite) AssertShootUpgraded(runtimeID string) {
	assert.True(s.t, s.provisionerClient.IsShootUpgraded(runtimeID), "The shoot %s expected to be upgraded", runtimeID)
}

func (s *OrchestrationSuite) AssertShootNotUpgraded(runtimeID string) {
	assert.False(s.t, s.provisionerClient.IsShootUpgraded(runtimeID), "The shoot %s expected to be not upgraded", runtimeID)
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

type ProvisioningSuite struct {
	provisionerClient   *provisioner.FakeClient
	provisioningManager *process.StagedManager
	provisioningQueue   *process.Queue
	storage             storage.BrokerStorage

	t         *testing.T
	k8sKcpCli client.Client
}

func (s *ProvisioningSuite) TearDown() {
	if r := recover(); r != nil {
		err := cleanupContainer()
		assert.NoError(s.t, err)
		panic(r)
	}
}

func NewProvisioningSuite(t *testing.T, multiZoneCluster bool, controlPlaneFailureTolerance string, useSmallerMachineTypes bool) *ProvisioningSuite {
	defer func() {
		if r := recover(); r != nil {
			err := cleanupContainer()
			assert.NoError(t, err)
			panic(r)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	logs := logrus.New()
	storageCleanup, db, err := GetStorageForE2ETests()
	assert.NoError(t, err)
	t.Cleanup(func() {
		defer cancel()
		if storageCleanup != nil {
			err := storageCleanup()
			assert.NoError(t, err)
		}
	})

	cfg := fixConfig()

	provisionerClient := provisioner.NewFakeClient()

	oidcDefaults := fixture.FixOIDCConfigDTO()

	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	additionalKymaVersions := []string{"1.19", "1.20", "main"}
	cli := fake.NewFakeClientWithScheme(sch, fixK8sResources(defaultKymaVer, additionalKymaVersions)...)
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logrus.New(), "keb-runtime-config"),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	inputFactory, err := input.NewInputBuilderFactory(configProvider, input.Config{
		MachineImageVersion:          "coreos",
		KubernetesVersion:            "1.18",
		MachineImage:                 "253",
		ProvisioningTimeout:          time.Minute,
		URL:                          "http://localhost",
		DefaultGardenerShootPurpose:  "testing",
		MultiZoneCluster:             multiZoneCluster,
		ControlPlaneFailureTolerance: controlPlaneFailureTolerance,
	}, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders, oidcDefaults, useSmallerMachineTypes)
	require.NoError(t, err)

	assert.NoError(t, err)

	edpClient := edp.NewFakeClient()

	accountProvider := fixAccountProvider()

	eventBroker := event.NewPubSub(logs)

	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Provisioning, logs.WithField("provisioning", "manager"))
	provisioningQueue := NewProvisioningProcessingQueue(ctx, provisionManager, workersAmount, cfg, db, provisionerClient, inputFactory, edpClient, accountProvider,
		kubeconfig.NewFakeK8sClientProvider(cli), cli, defaultOIDCValues(), logs)

	provisioningQueue.SpeedUp(10000)
	provisionManager.SpeedUp(10000)

	return &ProvisioningSuite{
		provisionerClient:   provisionerClient,
		provisioningManager: provisionManager,
		provisioningQueue:   provisioningQueue,
		storage:             db,
		k8sKcpCli:           cli,

		t: t,
	}
}

func (s *ProvisioningSuite) CreateProvisioning(options RuntimeOptions) string {
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: options.ProvidePlanID(),
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    options.ProvideSubAccountID(),
			UserID:          options.ProvideUserID(),
		},
		PlatformProvider: options.PlatformProvider,
		Parameters: internal.ProvisioningParametersDTO{
			Region:                options.ProvideRegion(),
			OIDC:                  options.ProvideOIDC(),
			RuntimeAdministrators: options.ProvideRuntimeAdmins(),
		},
		PlatformRegion: options.PlatformRegion,
	}

	shootName := gardener.CreateShootName()

	operation, err := internal.NewProvisioningOperationWithID(operationID, instanceID, provisioningParameters)
	require.NoError(s.t, err)
	operation.ShootName = shootName
	operation.ShootDomain = fmt.Sprintf("%s.%s.%s", shootName, "garden-dummy", strings.Trim("kyma.io", "."))
	operation.ShootDNSProviders = gardener.DNSProvidersData{}
	operation.DashboardURL = dashboardURL
	operation.State = orchestration.Pending

	err = s.storage.Operations().InsertOperation(operation.Operation)
	require.NoError(s.t, err)

	err = s.storage.Instances().Insert(internal.Instance{
		InstanceID:      instanceID,
		GlobalAccountID: globalAccountID,
		SubAccountID:    "dummy-sa",
		ServiceID:       provisioningParameters.ServiceID,
		ServiceName:     broker.KymaServiceName,
		ServicePlanID:   provisioningParameters.PlanID,
		ServicePlanName: broker.AzurePlanName,
		DashboardURL:    dashboardURL,
		Parameters:      operation.ProvisioningParameters,
	})

	s.provisioningQueue.Add(operation.ID)
	return operation.ID
}

func (s *ProvisioningSuite) CreateUnsuspension(options RuntimeOptions) string {
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: options.ProvidePlanID(),
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    options.ProvideSubAccountID(),
		},
		PlatformRegion: options.ProvidePlatformRegion(),
		Parameters: internal.ProvisioningParametersDTO{
			Region: options.ProvideRegion(),
		},
	}

	operation, err := internal.NewProvisioningOperationWithID(operationID, instanceID, provisioningParameters)
	operation.State = orchestration.Pending
	// in the real processing the URL is set in the handler
	operation.DashboardURL = dashboardURL
	require.NoError(s.t, err)

	err = s.storage.Operations().InsertOperation(operation.Operation)
	require.NoError(s.t, err)

	instance := &internal.Instance{
		InstanceID:      instanceID,
		GlobalAccountID: globalAccountID,
		SubAccountID:    "dummy-sa",
		ServiceID:       provisioningParameters.ServiceID,
		ServiceName:     broker.KymaServiceName,
		ServicePlanID:   provisioningParameters.PlanID,
		ServicePlanName: broker.AzurePlanName,
		DashboardURL:    dashboardURL,
		Parameters:      operation.ProvisioningParameters,
	}
	err = s.storage.Instances().Insert(*instance)

	suspensionOp := internal.NewSuspensionOperationWithID("susp-id", instance)
	suspensionOp.CreatedAt = time.Now().AddDate(0, 0, -10)
	suspensionOp.State = domain.Succeeded
	err = s.storage.Operations().InsertDeprovisioningOperation(suspensionOp)
	require.NoError(s.t, err)

	s.provisioningQueue.Add(operation.ID)
	return operation.ID
}

func (s *ProvisioningSuite) WaitForProvisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.Operation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetOperationByID(operationID)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *ProvisioningSuite) ProcessInfrastructureManagerProvisioningByRuntimeID(runtimeID string) {
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (bool, error) {
		gardenerCluster := &unstructured.Unstructured{}
		gardenerCluster.SetGroupVersionKind(steps.GardenerClusterGVK())
		err := s.k8sKcpCli.Get(context.Background(), client.ObjectKey{
			Namespace: "kyma-system",
			Name:      runtimeID,
		}, gardenerCluster)
		if err != nil {
			return false, nil
		}

		err = unstructured.SetNestedField(gardenerCluster.Object, "Ready", "status", "state")
		assert.NoError(s.t, err)
		err = s.k8sKcpCli.Update(context.Background(), gardenerCluster)
		return err == nil, nil
	})
	assert.NoError(s.t, err)
}

func (s *ProvisioningSuite) FinishProvisioningOperationByProvisioner(operationID string) {
	var op *internal.Operation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, op.RuntimeID)

	s.ProcessInfrastructureManagerProvisioningByRuntimeID(op.RuntimeID)
}

func (s *ProvisioningSuite) AssertProvisionerStartedProvisioning(operationID string) {
	// wait until ProvisioningOperation reaches CreateRuntime step
	var provisioningOp *internal.Operation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.storage.Operations().GetOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var status gqlschema.OperationStatus
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status = s.provisionerClient.FindInProgressOperationByRuntimeIDAndType(provisioningOp.RuntimeID, gqlschema.OperationTypeProvision)
		if status.ID != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, gqlschema.OperationStateInProgress, status.State)
}

func (s *ProvisioningSuite) AssertAllStagesFinished(operationID string) {
	operation, _ := s.storage.Operations().GetProvisioningOperationByID(operationID)
	steps := s.provisioningManager.GetAllStages()
	for _, stage := range steps {
		assert.True(s.t, operation.IsStageFinished(stage))
	}
}

func (s *ProvisioningSuite) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindInProgressOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, gqlschema.OperationStateSucceeded)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *ProvisioningSuite) AssertProvisioningRequest() {
	input := s.fetchProvisionInput()

	labels := input.RuntimeInput.Labels
	assert.Equal(s.t, instanceID, labels["broker_instance_id"])
	assert.Contains(s.t, labels, "global_subaccount_id")
	assert.NotEmpty(s.t, input.ClusterConfig.GardenerConfig.Name)
}

func (s *ProvisioningSuite) AssertProvider(provider string) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, provider, input.ClusterConfig.GardenerConfig.Provider)
}

func (s *ProvisioningSuite) fetchProvisionInput() gqlschema.ProvisionRuntimeInput {
	input := s.provisionerClient.GetLatestProvisionRuntimeInput()
	return input
}

func (s *ProvisioningSuite) AssertMinimalNumberOfNodes(nodes int) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, nodes, input.ClusterConfig.GardenerConfig.AutoScalerMin)
}

func (s *ProvisioningSuite) AssertMaximumNumberOfNodes(nodes int) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, nodes, input.ClusterConfig.GardenerConfig.AutoScalerMax)
}

func (s *ProvisioningSuite) AssertMachineType(machineType string) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, machineType, input.ClusterConfig.GardenerConfig.MachineType)
}

func (s *ProvisioningSuite) AssertOverrides(opID string, overrides []*gqlschema.ConfigEntryInput) {
	input := s.fetchProvisionInput()

	// values in arrays need to be sorted, because globalOverrides are coming from a map and map's elements' order is not deterministic
	sort.Slice(overrides, func(i, j int) bool {
		return overrides[i].Key < overrides[j].Key
	})
	sort.Slice(input.KymaConfig.Configuration, func(i, j int) bool {
		return input.KymaConfig.Configuration[i].Key < input.KymaConfig.Configuration[j].Key
	})

	assert.Equal(s.t, overrides, input.KymaConfig.Configuration)
}

func (s *ProvisioningSuite) AssertZonesCount(zonesCount *int, planID string) {
	provisionInput := s.fetchProvisionInput()

	switch planID {
	case broker.AzurePlanID:
		if zonesCount != nil {
			assert.Equal(s.t, *zonesCount, len(provisionInput.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones))
			break
		}
		assert.Equal(s.t, 1, len(provisionInput.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AzureConfig.AzureZones))
	case broker.AWSPlanID:
		if zonesCount != nil {
			assert.Equal(s.t, *zonesCount, len(provisionInput.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones))
			break
		}
		assert.Equal(s.t, 1, len(provisionInput.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones))
	case broker.GCPPlanID:
		if zonesCount != nil {
			assert.Equal(s.t, *zonesCount, len(provisionInput.ClusterConfig.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones))
			break
		}
		assert.Equal(s.t, 1, len(provisionInput.ClusterConfig.GardenerConfig.ProviderSpecificConfig.GcpConfig.Zones))
	default:
	}
}

func (s *ProvisioningSuite) AssertSubscription(shared bool, ht hyperscaler.Type) {
	input := s.fetchProvisionInput()
	secretName := input.ClusterConfig.GardenerConfig.TargetSecret
	if shared {
		assert.Equal(s.t, sharedSubscription(ht), secretName)
	} else {
		assert.Equal(s.t, regularSubscription(ht), secretName)
	}
}

func (s *ProvisioningSuite) AssertOIDC(oidcConfig gqlschema.OIDCConfigInput) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, &oidcConfig, input.ClusterConfig.GardenerConfig.OidcConfig)
}

func (s *ProvisioningSuite) AssertRuntimeAdmins(admins []string) {
	input := s.fetchProvisionInput()
	currentAdmins := input.ClusterConfig.Administrators

	assert.ElementsMatch(s.t, currentAdmins, admins)
}

func (s *ProvisioningSuite) AssertControlPlaneFailureTolerance(level string) {
	input := s.fetchProvisionInput()
	if level == "" {
		assert.Empty(s.t, input.ClusterConfig.GardenerConfig.ControlPlaneFailureTolerance)
	} else {
		require.NotNil(s.t, input.ClusterConfig.GardenerConfig.ControlPlaneFailureTolerance)
		assert.Equal(s.t, level, *input.ClusterConfig.GardenerConfig.ControlPlaneFailureTolerance)
	}
}

func regularSubscription(ht hyperscaler.Type) string {
	return fmt.Sprintf("regular-%s", ht.GetKey())
}

func sharedSubscription(ht hyperscaler.Type) string {
	return fmt.Sprintf("shared-%s", ht.GetKey())
}

func fixConfig() *Config {
	return &Config{
		DbInMemory:                         true,
		DisableProcessOperationsInProgress: false,
		DevelopmentMode:                    true,
		DumpProvisionerRequests:            true,
		OperationTimeout:                   2 * time.Minute,
		Provisioner: input.Config{
			ProvisioningTimeout:        2 * time.Minute,
			DeprovisioningTimeout:      2 * time.Minute,
			GardenerClusterStepTimeout: time.Second,
			MachineImage:               "gardenlinux",
			MachineImageVersion:        "12345.6",
			MultiZoneCluster:           true,
			RuntimeResourceStepTimeout: time.Minute,
			ClusterUpdateStepTimeout:   time.Minute,
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
			EnablePlans: []string{"azure", "trial", "aws", "own_cluster", "preview", "sap-converged-cloud", "gcp", "free"},
			Binding: broker.BindingConfig{
				Enabled:       true,
				BindablePlans: []string{"aws", "azure"},
			},
			AllowUpdateExpiredInstanceWithContext: true,
			KimConfig: broker.KimConfig{
				Enabled:      true,
				Plans:        []string{"preview"},
				KimOnlyPlans: []string{"preview"},
			},
		},
		TrialRegionMappingFilePath: "testdata/trial-regions.yaml",

		Notification: notification.Config{
			Url: "http://host:8080/",
		},
		OrchestrationConfig: kebOrchestration.Config{
			Namespace: "kcp-system",
			Name:      "orchestration-config",
		},
		MaxPaginationPage:                         100,
		FreemiumProviders:                         []string{"aws", "azure"},
		FreemiumWhitelistedGlobalAccountsFilePath: "testdata/freemium_whitelist.yaml",

		Provisioning:   process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		Deprovisioning: process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},
		Update:         process.StagedManagerConfiguration{MaxStepProcessingTime: time.Minute},

		ArchiveEnabled:                          true,
		CleaningEnabled:                         true,
		SapConvergedCloudRegionMappingsFilePath: "testdata/old-sap-converged-cloud-region-mappings.yaml",
		UpdateRuntimeResourceDelay:              time.Millisecond,
	}
}

func fixAccountProvider() *hyperscalerautomock.AccountProvider {
	accountProvider := hyperscalerautomock.AccountProvider{}

	accountProvider.On("GardenerSecretName", mock.Anything, mock.Anything, mock.Anything).Return(
		func(ht hyperscaler.Type, tn string, euaccess bool) string { return regularSubscription(ht) }, nil)

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
