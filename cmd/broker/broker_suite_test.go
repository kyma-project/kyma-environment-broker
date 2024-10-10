package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"code.cloudfoundry.org/lager"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	kebOrchestration "github.com/kyma-project/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	kebRuntime "github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const fixedGardenerNamespace = "garden-test"

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
	instanceName               = "my-service-instance"
	bindingName                = "my-binding"
	kymaNamespace              = "kyma-system"
)

var (
	serviceBindingGvk = schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    btpOperatorServiceBinding,
	}
	serviceInstanceGvk = schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    btpOperatorServiceInstance,
	}
)

// BrokerSuiteTest is a helper which allows to write simple tests of any KEB processes (provisioning, deprovisioning, update).
// The starting point of a test could be an HTTP call to Broker API.
type BrokerSuiteTest struct {
	db                storage.BrokerStorage
	storageCleanup    func() error
	provisionerClient *provisioner.FakeClient
	gardenerClient    dynamic.Interface

	httpServer *httptest.Server
	router     *mux.Router

	t                   *testing.T
	inputBuilderFactory input.CreatorForPlan

	k8sKcp client.Client
	k8sSKR client.Client

	poller broker.Poller

	eventBroker *event.PubSub
	metrics     *metricsv2.RegisterContainer
}

func (s *BrokerSuiteTest) TearDown() {
	if r := recover(); r != nil {
		err := cleanupContainer()
		assert.NoError(s.t, err)
		panic(r)
	}
	s.httpServer.Close()
	if s.storageCleanup != nil {
		err := s.storageCleanup()
		assert.NoError(s.t, err)
	}
}

func NewBrokerSuiteTest(t *testing.T, version ...string) *BrokerSuiteTest {
	cfg := fixConfig()
	return NewBrokerSuiteTestWithConfig(t, cfg, version...)
}

func NewBrokerSuiteTestWithConvergedCloudRegionMappings(t *testing.T, version ...string) *BrokerSuiteTest {
	cfg := fixConfig()
	cfg.SapConvergedCloudRegionMappingsFilePath = "testdata/sap-converged-cloud-region-mappings.yaml"
	return NewBrokerSuiteTestWithConfig(t, cfg, version...)
}

func NewBrokerSuitTestWithMetrics(t *testing.T, cfg *Config, version ...string) *BrokerSuiteTest {
	defer func() {
		if r := recover(); r != nil {
			err := cleanupContainer()
			assert.NoError(t, err)
			panic(r)
		}
	}()
	broker := NewBrokerSuiteTestWithConfig(t, cfg, version...)
	broker.metrics = metricsv2.Register(context.Background(), broker.eventBroker, broker.db.Operations(), broker.db.Instances(), cfg.MetricsV2, logrus.New())
	broker.router.Handle("/metrics", promhttp.Handler())
	return broker
}

func NewBrokerSuiteTestWithOptionalRegion(t *testing.T, version ...string) *BrokerSuiteTest {
	cfg := fixConfig()
	return NewBrokerSuiteTestWithConfig(t, cfg, version...)
}

func NewBrokerSuiteTestWithConfig(t *testing.T, cfg *Config, version ...string) *BrokerSuiteTest {
	defer func() {
		if r := recover(); r != nil {
			err := cleanupContainer()
			assert.NoError(t, err)
			panic(r)
		}
	}()
	ctx := context.Background()
	sch := internal.NewSchemeForTests(t)
	err := apiextensionsv1.AddToScheme(sch)
	require.NoError(t, err)
	err = imv1.AddToScheme(sch)
	require.NoError(t, err)
	additionalKymaVersions := []string{"1.19", "1.20", "main", "2.0"}
	additionalKymaVersions = append(additionalKymaVersions, version...)
	cli := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(fixK8sResources(defaultKymaVer, additionalKymaVersions)...).Build()

	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logrus.New(), "keb-runtime-config"),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())

	inputFactory, err := input.NewInputBuilderFactory(configProvider, input.Config{
		MachineImageVersion:          "253",
		KubernetesVersion:            "1.18",
		MachineImage:                 "coreos",
		URL:                          "http://localhost",
		DefaultGardenerShootPurpose:  "testing",
		DefaultTrialProvider:         internal.AWS,
		EnableShootAndSeedSameRegion: cfg.Provisioner.EnableShootAndSeedSameRegion,
	}, map[string]string{"cf-eu10": "europe", "cf-us10": "us"}, cfg.FreemiumProviders, defaultOIDCValues(), cfg.Broker.UseSmallerMachineTypes)

	storageCleanup, db, err := GetStorageForE2ETests()
	assert.NoError(t, err)

	require.NoError(t, err)

	logs := logrus.New()
	logs.SetLevel(logrus.DebugLevel)

	gardenerClient := gardener.NewDynamicFakeClient()

	provisionerClient := provisioner.NewFakeClientWithGardener(gardenerClient, "kcp-system")
	eventBroker := event.NewPubSub(logs)

	edpClient := edp.NewFakeClient()
	accountProvider := fixAccountProvider()
	require.NoError(t, err)

	fakeK8sSKRClient := fake.NewClientBuilder().WithScheme(sch).Build()
	k8sClientProvider := kubeconfig.NewFakeK8sClientProvider(fakeK8sSKRClient)
	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Provisioning, logs.WithField("provisioning", "manager"))
	provisioningQueue := NewProvisioningProcessingQueue(context.Background(), provisionManager, workersAmount, cfg, db, provisionerClient, inputFactory,
		edpClient, accountProvider, k8sClientProvider, cli, defaultOIDCValues(), logs)

	provisioningQueue.SpeedUp(10000)
	provisionManager.SpeedUp(10000)

	updateManager := process.NewStagedManager(db.Operations(), eventBroker, time.Hour, cfg.Update, logs)
	updateQueue := NewUpdateProcessingQueue(context.Background(), updateManager, 1, db, inputFactory, provisionerClient,
		eventBroker, *cfg, k8sClientProvider, cli, logs)
	updateQueue.SpeedUp(10000)
	updateManager.SpeedUp(10000)

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, time.Hour, cfg.Deprovisioning, logs.WithField("deprovisioning", "manager"))
	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db, eventBroker,
		provisionerClient, edpClient, accountProvider, k8sClientProvider, cli, configProvider, logs,
	)
	deprovisionManager.SpeedUp(10000)

	deprovisioningQueue.SpeedUp(10000)

	ts := &BrokerSuiteTest{
		db:                  db,
		storageCleanup:      storageCleanup,
		provisionerClient:   provisionerClient,
		gardenerClient:      gardenerClient,
		router:              mux.NewRouter(),
		t:                   t,
		inputBuilderFactory: inputFactory,
		k8sKcp:              cli,
		k8sSKR:              fakeK8sSKRClient,
		eventBroker:         eventBroker,
	}
	ts.poller = &broker.TimerPoller{PollInterval: 3 * time.Millisecond, PollTimeout: 3 * time.Second, Log: ts.t.Log}

	ts.CreateAPI(inputFactory, cfg, db, provisioningQueue, deprovisioningQueue, updateQueue, logs, k8sClientProvider, gardener.NewFakeClient())

	notificationFakeClient := notification.NewFakeClient()
	notificationBundleBuilder := notification.NewBundleBuilder(notificationFakeClient, cfg.Notification)

	runtimeLister := kebOrchestration.NewRuntimeLister(db.Instances(), db.Operations(), kebRuntime.NewConverter(defaultRegion), logs)
	runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient, fixedGardenerNamespace, runtimeLister, logs)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory, &upgrade_cluster.TimeSchedule{
		Retry:                 10 * time.Millisecond,
		StatusCheck:           100 * time.Millisecond,
		UpgradeClusterTimeout: 3 * time.Second,
	}, 250*time.Millisecond, runtimeResolver, notificationBundleBuilder, logs, cli, *cfg, 1000)

	clusterQueue.SpeedUp(1000)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, clusterQueue, cfg.MaxPaginationPage, logs)
	orchestrationHandler.AttachRoutes(ts.router)

	expirationHandler := expiration.NewHandler(db.Instances(), db.Operations(), deprovisioningQueue, logs)
	expirationHandler.AttachRoutes(ts.router)

	runtimeHandler := kebRuntime.NewHandler(db.Instances(), db.Operations(), db.RuntimeStates(), db.InstancesArchived(), db.Bindings(), cfg.MaxPaginationPage, cfg.DefaultRequestRegion, provisionerClient, cli, broker.KimConfig{
		Enabled: false,
	}, logs)
	runtimeHandler.AttachRoutes(ts.router)

	ts.httpServer = httptest.NewServer(ts.router)

	return ts
}

func fakeK8sClientProvider(k8sCli client.Client) func(s string) (client.Client, error) {
	return func(s string) (client.Client, error) {
		return k8sCli, nil
	}
}

func defaultOIDCValues() internal.OIDCConfigDTO {
	return internal.OIDCConfigDTO{
		ClientID:       "client-id-oidc",
		GroupsClaim:    "groups",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func defaultOIDCConfig() *gqlschema.OIDCConfigInput {
	return &gqlschema.OIDCConfigInput{
		ClientID:       defaultOIDCValues().ClientID,
		GroupsClaim:    defaultOIDCValues().GroupsClaim,
		IssuerURL:      defaultOIDCValues().IssuerURL,
		SigningAlgs:    defaultOIDCValues().SigningAlgs,
		UsernameClaim:  defaultOIDCValues().UsernameClaim,
		UsernamePrefix: defaultOIDCValues().UsernamePrefix,
	}
}

func (s *BrokerSuiteTest) ProcessInfrastructureManagerProvisioningByRuntimeID(runtimeID string) {
	err := s.poller.Invoke(func() (bool, error) {
		gardenerCluster := &unstructured.Unstructured{}
		gardenerCluster.SetGroupVersionKind(steps.GardenerClusterGVK())
		err := s.k8sKcp.Get(context.Background(), client.ObjectKey{
			Namespace: "kyma-system",
			Name:      runtimeID,
		}, gardenerCluster)
		if err != nil {
			return false, nil
		}

		err = unstructured.SetNestedField(gardenerCluster.Object, "Ready", "status", "state")
		assert.NoError(s.t, err)
		err = s.k8sKcp.Update(context.Background(), gardenerCluster)
		return err == nil, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) ChangeDefaultTrialProvider(provider internal.CloudProvider) {
	s.inputBuilderFactory.(*input.InputBuilderFactory).SetDefaultTrialProvider(provider)
}

func (s *BrokerSuiteTest) CallAPI(method string, path string, body string) *http.Response {
	cli := s.httpServer.Client()
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", s.httpServer.URL, path), bytes.NewBuffer([]byte(body)))
	req.Header.Set("X-Broker-API-Version", "2.15")
	require.NoError(s.t, err)

	resp, err := cli.Do(req)
	require.NoError(s.t, err)
	return resp
}

func (s *BrokerSuiteTest) CreateAPI(inputFactory broker.PlanValidator, cfg *Config, db storage.BrokerStorage, provisioningQueue *process.Queue, deprovisionQueue *process.Queue, updateQueue *process.Queue, logs logrus.FieldLogger, skrK8sClientProvider *kubeconfig.FakeProvider, gardenerClient client.Client) {
	servicesConfig := map[string]broker.Service{
		broker.KymaServiceName: {
			Description: "",
			Metadata: broker.ServiceMetadata{
				DisplayName: "kyma",
				SupportUrl:  "https://kyma-project.io",
			},
			Plans: map[string]broker.PlanData{
				broker.AzurePlanID: {
					Description: broker.AzurePlanName,
					Metadata:    broker.PlanMetadata{},
				},
				broker.AWSPlanName: {
					Description: broker.AWSPlanName,
					Metadata:    broker.PlanMetadata{},
				},
				broker.SapConvergedCloudPlanName: {
					Description: broker.SapConvergedCloudPlanName,
					Metadata:    broker.PlanMetadata{},
				},
			},
		},
	}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	var fakeKcpK8sClient = fake.NewClientBuilder().Build()
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("Build", nil).Return("--kubeconfig file", nil)
	createAPI(s.router, servicesConfig, inputFactory, cfg, db, provisioningQueue, deprovisionQueue, updateQueue, lager.NewLogger("api"), logs, planDefaults, kcBuilder, skrK8sClientProvider, skrK8sClientProvider, gardenerClient, fakeKcpK8sClient)

	s.httpServer = httptest.NewServer(s.router)
}

func (s *BrokerSuiteTest) CreateProvisionedRuntime(options RuntimeOptions) string {
	randomInstanceId := uuid.New().String()

	instance := fixture.FixInstance(randomInstanceId)
	instance.GlobalAccountID = options.ProvideGlobalAccountID()
	instance.SubAccountID = options.ProvideSubAccountID()
	instance.InstanceDetails.SubAccountID = options.ProvideSubAccountID()
	instance.Parameters.PlatformRegion = options.ProvidePlatformRegion()
	instance.Parameters.Parameters.Region = options.ProvideRegion()
	instance.ProviderRegion = *options.ProvideRegion()

	provisioningOperation := fixture.FixProvisioningOperation(operationID, randomInstanceId)

	require.NoError(s.t, s.db.Instances().Insert(instance))
	require.NoError(s.t, s.db.Operations().InsertOperation(provisioningOperation))

	return instance.InstanceID
}

func (s *BrokerSuiteTest) WaitForProvisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.ProvisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, err = s.db.Operations().GetProvisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *BrokerSuiteTest) WaitForOperationState(operationID string, state domain.LastOperationState) {
	var op *internal.Operation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, err = s.db.Operations().GetOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s != %s. The existing operation %+v", state, op.State, op)
}

func (s *BrokerSuiteTest) GetOperation(operationID string) *internal.Operation {
	var op *internal.Operation
	_ = s.poller.Invoke(func() (done bool, err error) {
		op, err = s.db.Operations().GetOperationByID(operationID)
		return err != nil, nil
	})

	return op
}

func (s *BrokerSuiteTest) WaitForLastOperation(iid string, state domain.LastOperationState) string {
	var op *internal.Operation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, _ = s.db.Operations().GetLastOperation(iid)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)

	return op.ID
}

func (s *BrokerSuiteTest) LastOperation(iid string) *internal.Operation {
	op, _ := s.db.Operations().GetLastOperation(iid)
	return op
}

func (s *BrokerSuiteTest) FinishProvisioningOperationByProvisionerAndInfrastructureManager(operationID string, operationState gqlschema.OperationState) {
	var op *internal.ProvisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, operationState, op.RuntimeID)
	if operationState == gqlschema.OperationStateSucceeded {
		s.ProcessInfrastructureManagerProvisioningByRuntimeID(op.RuntimeID)
	}
}

func (s *BrokerSuiteTest) FailProvisioningOperationByProvisioner(operationID string) {
	var op *internal.ProvisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, gqlschema.OperationStateFailed, op.RuntimeID)
}

func (s *BrokerSuiteTest) FailDeprovisioningOperationByProvisioner(operationID string) {
	var op *internal.DeprovisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, _ = s.db.Operations().GetDeprovisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeDeprovision, gqlschema.OperationStateFailed, op.RuntimeID)
}

func (s *BrokerSuiteTest) FinishDeprovisioningOperationByProvisioner(operationID string) {
	var op *internal.DeprovisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, err = s.db.Operations().GetDeprovisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	err = s.gardenerClient.Resource(gardener.ShootResource).
		Namespace(fixedGardenerNamespace).
		Delete(context.Background(), op.ShootName, v1.DeleteOptions{})

	s.finishOperationByProvisioner(gqlschema.OperationTypeDeprovision, gqlschema.OperationStateSucceeded, op.RuntimeID)
}

func (s *BrokerSuiteTest) FinishUpdatingOperationByProvisioner(operationID string) {
	var op *internal.Operation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, _ = s.db.Operations().GetOperationByID(operationID)
		if op == nil || op.RuntimeID == "" || op.ProvisionerOperationID == "" {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)
	s.finishOperationByOpIDByProvisioner(gqlschema.OperationTypeUpgradeShoot, gqlschema.OperationStateSucceeded, op.ID)
}

func (s *BrokerSuiteTest) FinishDeprovisioningOperationByProvisionerForGivenOpId(operationID string) {
	var op *internal.DeprovisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, err = s.db.Operations().GetDeprovisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.RuntimeID != "" && op.ProvisionerOperationID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	uns, err := s.gardenerClient.Resource(gardener.ShootResource).
		Namespace(fixedGardenerNamespace).
		List(context.Background(), v1.ListOptions{})
	require.NoError(s.t, err)
	if len(uns.Items) == 0 {
		s.Log(fmt.Sprintf("shoot %s doesn't exist", op.ShootName))
		s.finishOperationByOpIDByProvisioner(gqlschema.OperationTypeDeprovision, gqlschema.OperationStateSucceeded, op.ID)
		return
	}

	err = s.gardenerClient.Resource(gardener.ShootResource).
		Namespace(fixedGardenerNamespace).
		Delete(context.Background(), op.ShootName, v1.DeleteOptions{})
	require.NoError(s.t, err)

	s.finishOperationByOpIDByProvisioner(gqlschema.OperationTypeDeprovision, gqlschema.OperationStateSucceeded, op.ID)
}

func (s *BrokerSuiteTest) waitForRuntimeAndMakeItReady(id string) {
	var op *internal.Operation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, err = s.db.Operations().GetOperationByID(id)
		if err != nil {
			return false, nil
		}
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID")

	runtimeID := op.RuntimeID

	var runtime imv1.Runtime
	err = s.poller.Invoke(func() (done bool, err error) {
		e := s.k8sKcp.Get(context.Background(), client.ObjectKey{Namespace: "kyma-system", Name: runtimeID}, &runtime)
		if e == nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the runtime to be created")

	runtime.Status.State = imv1.RuntimeStateReady
	err = s.k8sKcp.Update(context.Background(), &runtime)
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) finishOperationByProvisioner(operationType gqlschema.OperationType, state gqlschema.OperationState, runtimeID string) {
	err := s.poller.Invoke(func() (bool, error) {
		status := s.provisionerClient.FindInProgressOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, state)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *BrokerSuiteTest) finishOperationByOpIDByProvisioner(operationType gqlschema.OperationType, state gqlschema.OperationState, operationID string) {
	err := s.poller.Invoke(func() (bool, error) {
		op, err := s.db.Operations().GetOperationByID(operationID)
		if err != nil {
			s.Log(fmt.Sprintf("failed to GetOperationsByID: %v", err))
			return false, nil
		}
		status, err := s.provisionerClient.RuntimeOperationStatus("", op.ProvisionerOperationID)
		if err != nil {
			s.Log(fmt.Sprintf("failed to get RuntimeOperationStatus: %v", err))
			return false, nil
		}
		if status.Operation != operationType {
			s.Log(fmt.Sprintf("operation types don't match, expected: %s, actual: %s", operationType.String(), status.Operation.String()))
			return false, nil
		}
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, state)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *BrokerSuiteTest) AssertProvisionerStartedProvisioning(operationID string) {
	// wait until ProvisioningOperation reaches CreateRuntime step
	var provisioningOp *internal.ProvisioningOperation
	err := s.poller.Invoke(func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	require.NotNil(s.t, provisioningOp, "Provisioning operation should not be nil")

	var status gqlschema.OperationStatus
	err = s.poller.Invoke(func() (bool, error) {
		status = s.provisionerClient.FindInProgressOperationByRuntimeIDAndType(provisioningOp.RuntimeID, gqlschema.OperationTypeProvision)
		if status.ID != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, gqlschema.OperationStateInProgress, status.State)
}

func (s *BrokerSuiteTest) FinishUpgradeClusterOperationByProvisioner(operationID string) {
	var upgradeOp *internal.UpgradeClusterOperation
	err := s.poller.Invoke(func() (bool, error) {
		op, err := s.db.Operations().GetUpgradeClusterOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		upgradeOp = op
		return true, nil
	})
	assert.NoError(s.t, err)

	s.finishOperationByOpIDByProvisioner(gqlschema.OperationTypeUpgradeShoot, gqlschema.OperationStateSucceeded, upgradeOp.Operation.ID)
}

func (s *BrokerSuiteTest) DecodeErrorResponse(resp *http.Response) apiresponses.ErrorResponse {
	m, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(s.t, err)

	r := apiresponses.ErrorResponse{}
	err = json.Unmarshal(m, &r)
	require.NoError(s.t, err)

	return r
}

func (s *BrokerSuiteTest) ReadResponse(resp *http.Response) []byte {
	m, err := io.ReadAll(resp.Body)
	s.Log(string(m))
	require.NoError(s.t, err)
	return m
}

func (s *BrokerSuiteTest) DecodeOperationID(resp *http.Response) string {
	m := s.ReadResponse(resp)
	var provisioningResp struct {
		Operation string `json:"operation"`
	}
	err := json.Unmarshal(m, &provisioningResp)
	require.NoError(s.t, err)

	return provisioningResp.Operation
}

func (s *BrokerSuiteTest) DecodeOrchestrationID(resp *http.Response) string {
	m, err := io.ReadAll(resp.Body)
	s.Log(string(m))
	require.NoError(s.t, err)
	var upgradeResponse orchestration.UpgradeResponse
	err = json.Unmarshal(m, &upgradeResponse)
	require.NoError(s.t, err)

	return upgradeResponse.OrchestrationID
}

func (s *BrokerSuiteTest) DecodeLastUpgradeKymaOperationFromOrchestration(resp *http.Response) (*orchestration.OperationResponse, error) {
	m, err := io.ReadAll(resp.Body)
	s.Log(string(m))
	require.NoError(s.t, err)
	var operationsList orchestration.OperationResponseList
	err = json.Unmarshal(m, &operationsList)
	require.NoError(s.t, err)

	if operationsList.TotalCount == 0 || len(operationsList.Data) == 0 {
		return nil, errors.New("no operations found for given orchestration")
	}

	return &operationsList.Data[len(operationsList.Data)-1], nil
}

func (s *BrokerSuiteTest) DecodeLastUpgradeKymaOperationIDFromOrchestration(resp *http.Response) (string, error) {
	operation, err := s.DecodeLastUpgradeKymaOperationFromOrchestration(resp)
	if err == nil {
		return operation.OperationID, err
	} else {
		return "", err
	}
}

func (s *BrokerSuiteTest) DecodeLastUpgradeClusterOperationIDFromOrchestration(orchestrationID string) (string, error) {
	var operationsList orchestration.OperationResponseList
	err := s.poller.Invoke(func() (bool, error) {
		resp := s.CallAPI("GET", fmt.Sprintf("orchestrations/%s/operations", orchestrationID), "")
		m, err := io.ReadAll(resp.Body)
		s.Log(string(m))
		if err != nil {
			return false, fmt.Errorf("failed to read response body: %v", err)
		}
		operationsList = orchestration.OperationResponseList{}
		err = json.Unmarshal(m, &operationsList)
		if err != nil {
			return false, fmt.Errorf("failed to marshal: %v", err)
		}
		if operationsList.TotalCount == 0 || len(operationsList.Data) == 0 {
			return false, nil
		}
		return true, nil
	})
	require.NoError(s.t, err)
	if operationsList.TotalCount == 0 || len(operationsList.Data) == 0 {
		return "", errors.New("no operations found for given orchestration")
	}

	return operationsList.Data[len(operationsList.Data)-1].OperationID, nil
}

func (s *BrokerSuiteTest) AssertShootUpgrade(operationID string, config gqlschema.UpgradeShootInput) {
	// wait until the operation reaches the call to a Provisioner (provisioner operation ID is stored)
	var provisioningOp *internal.Operation
	err := s.poller.Invoke(func() (bool, error) {
		op, err := s.db.Operations().GetOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" || broker.IsOwnClusterPlan(op.ProvisioningParameters.PlanID) {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	require.NoError(s.t, err)

	var shootUpgrade gqlschema.UpgradeShootInput
	var found bool
	err = s.poller.Invoke(func() (bool, error) {
		shootUpgrade, found = s.provisionerClient.LastShootUpgrade(provisioningOp.RuntimeID)
		if found {
			return true, nil
		}
		return false, nil
	})
	require.NoError(s.t, err)

	assert.Equal(s.t, config, shootUpgrade)
}

func (s *BrokerSuiteTest) AssertInstanceRuntimeAdmins(instanceId string, expectedAdmins []string) {
	var instance *internal.Instance
	err := s.poller.Invoke(func() (bool, error) {
		instance = s.GetInstance(instanceId)
		if instance != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, expectedAdmins, instance.Parameters.Parameters.RuntimeAdministrators)
}

func (s *BrokerSuiteTest) fetchProvisionInput() gqlschema.ProvisionRuntimeInput {
	input := s.provisionerClient.GetLatestProvisionRuntimeInput()
	return input
}

func (s *BrokerSuiteTest) AssertProvider(expectedProvider string) {
	input := s.fetchProvisionInput()
	assert.Equal(s.t, expectedProvider, input.ClusterConfig.GardenerConfig.Provider)
}

func (s *BrokerSuiteTest) AssertProvisionRuntimeInputWithoutKymaConfig() {
	input := s.fetchProvisionInput()
	assert.Nil(s.t, input.KymaConfig)
}

func (s *BrokerSuiteTest) AssertDisabledNetworkFilterForProvisioning(val *bool) {
	var got, exp string
	err := s.poller.Invoke(func() (bool, error) {
		input := s.provisionerClient.GetLatestProvisionRuntimeInput()
		gc := input.ClusterConfig.GardenerConfig
		if reflect.DeepEqual(val, gc.ShootNetworkingFilterDisabled) {
			return true, nil
		}
		got = "<nil>"
		if gc.ShootNetworkingFilterDisabled != nil {
			got = fmt.Sprintf("%v", *gc.ShootNetworkingFilterDisabled)
		}
		exp = "<nil>"
		if val != nil {
			exp = fmt.Sprintf("%v", *val)
		}
		return false, nil
	})
	if err != nil {
		err = fmt.Errorf("ShootNetworkingFilterDisabled expected %v, got %v", exp, got)
	}
	require.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertDisabledNetworkFilterRuntimeState(runtimeid, op string, val *bool) {
	var got, exp string
	err := s.poller.Invoke(func() (bool, error) {
		states, _ := s.db.RuntimeStates().ListByRuntimeID(runtimeid)
		exp = "<nil>"
		if val != nil {
			exp = fmt.Sprintf("%v", *val)
		}
		for _, rs := range states {
			if rs.OperationID != op {
				// skip runtime states for different operations
				continue
			}
			if reflect.DeepEqual(val, rs.ClusterConfig.ShootNetworkingFilterDisabled) {
				return true, nil
			}
			got = "<nil>"
			if rs.ClusterConfig.ShootNetworkingFilterDisabled != nil {
				got = fmt.Sprintf("%v", *rs.ClusterConfig.ShootNetworkingFilterDisabled)
			}
			return false, nil
		}
		return false, nil
	})
	if err != nil {
		err = fmt.Errorf("ShootNetworkingFilterDisabled expected %v, got %v", exp, got)
	}
	require.NoError(s.t, err)
}

func (s *BrokerSuiteTest) LastProvisionInput(iid string) gqlschema.ProvisionRuntimeInput {
	// wait until the operation reaches the call to a Provisioner (provisioner operation ID is stored)
	err := s.poller.Invoke(func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByInstanceID(iid)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	return s.provisionerClient.LastProvisioning()
}

func (s *BrokerSuiteTest) Log(msg string) {
	s.t.Log(msg)
}

func (s *BrokerSuiteTest) EnableDumpingProvisionerRequests() {
	s.provisionerClient.EnableRequestDumping()
}

func (s *BrokerSuiteTest) GetInstance(iid string) *internal.Instance {
	inst, err := s.db.Instances().GetByID(iid)
	require.NoError(s.t, err)
	return inst
}

func (s *BrokerSuiteTest) processProvisioningByOperationID(opID string) {
	s.WaitForProvisioningState(opID, domain.InProgress)
	s.AssertProvisionerStartedProvisioning(opID)

	s.FinishProvisioningOperationByProvisionerAndInfrastructureManager(opID, gqlschema.OperationStateSucceeded)
	_, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(fixedGardenerNamespace).Create(context.Background(), s.fixGardenerShootForOperationID(opID), v1.CreateOptions{})
	require.NoError(s.t, err)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) processUpdatingByOperationID(opID string) {
	s.WaitForProvisioningState(opID, domain.InProgress)

	s.FinishUpdatingOperationByProvisioner(opID)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) failProvisioningByOperationID(opID string) {
	s.WaitForProvisioningState(opID, domain.InProgress)
	s.AssertProvisionerStartedProvisioning(opID)

	s.FinishProvisioningOperationByProvisionerAndInfrastructureManager(opID, gqlschema.OperationStateFailed)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Failed)
}

func (s *BrokerSuiteTest) fixGardenerShootForOperationID(opID string) *unstructured.Unstructured {
	op, err := s.db.Operations().GetProvisioningOperationByID(opID)
	require.NoError(s.t, err)

	un := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      op.ShootName,
				"namespace": fixedGardenerNamespace,
				"labels": map[string]interface{}{
					globalAccountLabel: op.ProvisioningParameters.ErsContext.GlobalAccountID,
					subAccountLabel:    op.ProvisioningParameters.ErsContext.SubAccountID,
				},
				"annotations": map[string]interface{}{
					runtimeIDAnnotation: op.RuntimeID,
				},
			},
			"spec": map[string]interface{}{
				"region": "eu",
				"maintenance": map[string]interface{}{
					"timeWindow": map[string]interface{}{
						"begin": "030000+0000",
						"end":   "040000+0000",
					},
				},
			},
		},
	}
	un.SetGroupVersionKind(shootGVK)
	return &un
}

func (s *BrokerSuiteTest) processProvisioningByInstanceID(iid string) {
	opID := s.WaitForLastOperation(iid, domain.InProgress)

	s.processProvisioningByOperationID(opID)
}

func (s *BrokerSuiteTest) AssertAWSRegionAndZone(region string) {
	input := s.provisionerClient.GetLatestProvisionRuntimeInput()
	assert.Equal(s.t, region, input.ClusterConfig.GardenerConfig.Region)
	assert.Contains(s.t, input.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[0].Name, region)
}

func (s *BrokerSuiteTest) AssertAzureRegion(region string) {
	input := s.provisionerClient.GetLatestProvisionRuntimeInput()
	assert.Equal(s.t, region, input.ClusterConfig.GardenerConfig.Region)
}

func (s *BrokerSuiteTest) AssertKymaResourceExists(opId string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)

	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)

	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertKymaResourceExistsByInstanceID(instanceID string) {
	instance := s.GetInstance(instanceID)

	obj := &unstructured.Unstructured{}
	obj.SetName(instance.RuntimeID)
	obj.SetNamespace("kyma-system")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})

	err := s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)

	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertKymaResourceNotExists(opId string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)

	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)

	assert.Error(s.t, err)
}

func (s *BrokerSuiteTest) AssertKymaAnnotationExists(opId, annotationName string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)
	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)

	assert.Contains(s.t, obj.GetAnnotations(), annotationName)
}

func (s *BrokerSuiteTest) AssertKymaLabelsExist(opId string, expectedLabels map[string]string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)
	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)

	assert.Subset(s.t, obj.GetLabels(), expectedLabels)
}

func (s *BrokerSuiteTest) AssertKymaLabelNotExists(opId string, notExpectedLabel string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)
	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	})

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)

	assert.NotContains(s.t, obj.GetLabels(), notExpectedLabel)
}

func (s *BrokerSuiteTest) fixServiceBindingAndInstances(t *testing.T) {
	createResource(t, serviceInstanceGvk, s.k8sSKR, kymaNamespace, instanceName)
	createResource(t, serviceBindingGvk, s.k8sSKR, kymaNamespace, bindingName)
}

func (s *BrokerSuiteTest) assertServiceBindingAndInstancesAreRemoved(t *testing.T) {
	assertResourcesAreRemoved(t, serviceInstanceGvk, s.k8sSKR)
	assertResourcesAreRemoved(t, serviceBindingGvk, s.k8sSKR)
}

func (s *BrokerSuiteTest) WaitForInstanceArchivedCreated(iid string) {

	err := s.poller.Invoke(func() (bool, error) {
		_, err := s.db.InstancesArchived().GetByInstanceID(iid)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	assert.NoError(s.t, err)

}

func (s *BrokerSuiteTest) WaitForOperationsNotExists(iid string) {
	err := s.poller.Invoke(func() (bool, error) {
		ops, err := s.db.Operations().ListOperationsByInstanceID(iid)
		if err != nil {
			return false, nil
		}

		return len(ops) == 0, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) WaitFor(f func() bool) {
	err := s.poller.Invoke(func() (bool, error) {
		if f() {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) ParseLastOperationResponse(resp *http.Response) domain.LastOperation {
	data, err := io.ReadAll(resp.Body)
	assert.NoError(s.t, err)
	var operationResponse domain.LastOperation
	err = json.Unmarshal(data, &operationResponse)
	assert.NoError(s.t, err)
	return operationResponse
}

func (s *BrokerSuiteTest) AssertMetric(operationType internal.OperationType, state domain.LastOperationState, plan string, expected int) {
	metric, err := s.metrics.OperationStats.Metric(operationType, state, broker.PlanID(plan))
	assert.NoError(s.t, err)
	assert.NotNil(s.t, metric)
	assert.Equal(s.t, float64(expected), testutil.ToFloat64(metric), fmt.Sprintf("expected %s metric for %s plan to be %d", operationType, plan, expected))
}

func (s *BrokerSuiteTest) AssertMetrics2(expected int, operation internal.Operation) {
	if expected == 0 && operation.ID == "" {
		assert.Truef(s.t, true, "expected 0 metrics for operation %s", operation.ID)
		return
	}
	a := s.metrics.OperationResult.Metrics().With(metricsv2.GetLabels(operation))
	assert.NotNil(s.t, a)
	assert.Equal(s.t, float64(expected), testutil.ToFloat64(a))
}

func (s *BrokerSuiteTest) GetRuntimeResourceByInstanceID(iid string) imv1.Runtime {
	var runtimes imv1.RuntimeList
	err := s.k8sKcp.List(context.Background(), &runtimes, client.MatchingLabels{"kyma-project.io/instance-id": iid})
	require.NoError(s.t, err)
	require.Equal(s.t, 1, len(runtimes.Items))
	return runtimes.Items[0]
}

func assertResourcesAreRemoved(t *testing.T, gvk schema.GroupVersionKind, k8sClient client.Client) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err := k8sClient.List(context.TODO(), list)
	assert.NoError(t, err)
	assert.Zero(t, len(list.Items))
}

func createResource(t *testing.T, gvk schema.GroupVersionKind, k8sClient client.Client, namespace string, name string) {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace(namespace)
	object.SetName(name)
	err := k8sClient.Create(context.TODO(), object)
	assert.NoError(t, err)
}
