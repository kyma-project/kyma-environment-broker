package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	dynamicFake "k8s.io/client-go/dynamic/fake"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/customresources"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	kebRuntime "github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	"code.cloudfoundry.org/lager"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/google/uuid"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
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
	testSuiteSpeedUpFactor     = 10000
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
	db             storage.BrokerStorage
	storageCleanup func() error
	gardenerClient dynamic.Interface

	httpServer *httptest.Server
	router     *httputil.Router

	t *testing.T

	k8sKcp client.Client
	k8sSKR client.Client

	poller broker.Poller

	eventBroker              *event.PubSub
	metrics                  *metrics.RegisterContainer
	k8sDeletionObjectTracker Deleter
}

func (s *BrokerSuiteTest) AddNotCompletedStep(suspensionOpID string) {
	op, err := s.db.Operations().GetOperationByID(suspensionOpID)
	require.NoError(s.t, err)
	op.ExcutedButNotCompleted = append(op.ExcutedButNotCompleted, "Simulated not completed step")
	_, err = s.db.Operations().UpdateOperation(*op)
	require.NoError(s.t, err)
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

func NewBrokerSuitTestWithMetrics(t *testing.T, cfg *Config, version ...string) *BrokerSuiteTest {
	defer func() {
		if r := recover(); r != nil {
			err := cleanupContainer()
			assert.NoError(t, err)
			panic(r)
		}
	}()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	broker := NewBrokerSuiteTestWithConfig(t, cfg, version...)
	broker.metrics = metrics.Register(context.Background(), broker.eventBroker, broker.db, cfg.Metrics, log)
	broker.router.Handle("/metrics", promhttp.Handler())
	return broker
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

	ot := NewTestingObjectTracker(sch)
	cli := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(fixK8sResources(defaultKymaVer, additionalKymaVersions)...).
		WithObjectTracker(ot).Build()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, log),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())

	storageCleanup, db, err := GetStorageForE2ETests()
	assert.NoError(t, err)

	require.NoError(t, err)

	gardenerClient := gardener.NewDynamicFakeClient(fixSecrets()...)

	eventBroker := event.NewPubSub(log)

	createSubscriptions(t, gardenerClient, cfg.SubscriptionGardenerResource)
	require.NoError(t, err)

	gardenerClientWithNamespace := gardener.NewClient(gardenerClient, gardenerKymaNamespace)

	providerSpec, err := configuration.NewProviderSpecFromFile(cfg.ProvidersConfigurationFilePath)
	fatalOnError(err, log)
	plansSpec, err := configuration.NewPlanSpecificationsFromFile(cfg.PlansConfigurationFilePath)
	fatalOnError(err, log)
	defaultOIDC := defaultOIDCValues()
	runtimeConfigProvider := kebConfig.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, kebConfig.RuntimeConfigurationRequiredFields)
	channelResolver, err := kebConfig.NewChannelResolver(runtimeConfigProvider, broker.AllPlanNames(), log)
	fatalOnError(err, log)
	schemaService := broker.NewSchemaService(providerSpec, plansSpec, &defaultOIDC, cfg.Broker, cfg.InfrastructureManager.IngressFilteringPlans, channelResolver)
	fatalOnError(err, log)

	fakeK8sSKRClient := fake.NewClientBuilder().WithScheme(sch).Build()
	k8sClientProvider := kubeconfig.NewFakeK8sClientProvider(fakeK8sSKRClient)
	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.Broker.OperationTimeout, cfg.Provisioning, log.With("provisioning", "manager"))

	rulesService, err := rules.NewRulesServiceFromFile("testdata/hap-rules.yaml", sets.New(maps.Keys(broker.PlanIDsMapping)...), sets.New([]string(cfg.Broker.EnablePlans)...).Delete("own_cluster"))
	require.NoError(t, err)
	if rulesService.ValidationInfo != nil {
		require.Empty(t, rulesService.ValidationInfo.PlanErrors)
	}

	awsClientFactory := fixture.NewFakeAWSClientFactory(fixDiscoveredZones(), nil)

	provisioningQueue := NewProvisioningProcessingQueue(context.Background(), provisionManager, workersAmount, cfg, db, configProvider,
		k8sClientProvider, cli, gardenerClientWithNamespace, defaultOIDCValues(), log, rulesService,
		workersProvider(cfg.InfrastructureManager, providerSpec), providerSpec, awsClientFactory)

	provisioningQueue.SpeedUp(testSuiteSpeedUpFactor)
	provisionManager.SpeedUp(testSuiteSpeedUpFactor)

	updateManager := process.NewStagedManager(db.Operations(), eventBroker, time.Hour, cfg.Update, log.With("update", "manager"))
	updateQueue := NewUpdateProcessingQueue(context.Background(), updateManager, 1, db, *cfg, cli, log, workersProvider(cfg.InfrastructureManager, providerSpec),
		schemaService, plansSpec, configProvider, providerSpec, gardenerClientWithNamespace, awsClientFactory)
	updateQueue.SpeedUp(testSuiteSpeedUpFactor)
	updateManager.SpeedUp(testSuiteSpeedUpFactor)

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, time.Hour, cfg.Deprovisioning, log.With("deprovisioning", "manager"))

	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db,
		k8sClientProvider, cli, configProvider, gardenerClient, "kyma", log)
	deprovisionManager.SpeedUp(testSuiteSpeedUpFactor)

	deprovisioningQueue.SpeedUp(testSuiteSpeedUpFactor)

	ts := &BrokerSuiteTest{
		db:             db,
		storageCleanup: storageCleanup,
		gardenerClient: gardenerClient,
		router:         httputil.NewRouter(),
		t:              t,
		k8sKcp:         cli,
		k8sSKR:         fakeK8sSKRClient,
		eventBroker:    eventBroker,

		k8sDeletionObjectTracker: ot,
	}
	ts.poller = &broker.TimerPoller{PollInterval: 3 * time.Millisecond, PollTimeout: 800 * time.Millisecond, Log: ts.t.Log}

	ts.CreateAPI(cfg, db, provisioningQueue, deprovisioningQueue, updateQueue, log, k8sClientProvider, eventBroker, configProvider, plansSpec, rulesService, gardenerClientWithNamespace, awsClientFactory)

	expirationHandler := expiration.NewHandler(db.Instances(), db.Operations(), deprovisioningQueue, log)
	expirationHandler.AttachRoutes(ts.router)

	runtimeHandler := kebRuntime.NewHandler(db, cfg.MaxPaginationPage, cfg.Broker.DefaultRequestRegion, cli, log)
	runtimeHandler.AttachRoutes(ts.router)

	ts.httpServer = httptest.NewServer(ts.router)

	return ts
}

func (s *BrokerSuiteTest) GetKcpClient() client.Client {
	return s.k8sKcp
}

func createSubscriptions(t *testing.T, gardenerClient *dynamicFake.FakeDynamicClient, bindingResource string) {
	resource := gardener.SecretBindingResource
	if strings.ToLower(bindingResource) == "credentialsbinding" {
		resource = gardener.CredentialsBindingResource
	}

	for sbName, labels := range map[string]map[string]string{
		"sb-azure": {
			"hyperscalerType": "azure",
		},
		"sb-aws": {
			"hyperscalerType": "aws",
		},
		"sb-gcp": {
			"hyperscalerType": "gcp",
		},
		"sb-gcp_cf-sa30": {
			"hyperscalerType": "gcp_cf-sa30",
		},
		"sb-aws-shared": {
			"hyperscalerType": "aws",
			"shared":          "true",
		},
		"sb-azure-shared": {
			"hyperscalerType": "azure",
			"shared":          "true",
		},
		"sb-aws-eu-access": {
			"hyperscalerType": "aws",
			"euAccess":        "true",
		},
		"sb-azure-eu-access": {
			"hyperscalerType": "azure",
			"euAccess":        "true",
		},
		"sb-gcp-ksa": {
			"hyperscalerType": "gcp-cf-sa30",
		},
		"sb-openstack_eu-de-1": {
			"hyperscalerType": "openstack_eu-de-1",
			"shared":          "true",
		},
		"sb-openstack_eu-de-2": {
			"hyperscalerType": "openstack_eu-de-2",
			"shared":          "true",
		},
		"sb-alicloud": {
			"hyperscalerType": "alicloud",
		},
	} {

		var unstructuredObj *unstructured.Unstructured
		if resource == gardener.SecretBindingResource {
			sb := &gardener.SecretBinding{}
			sb.SetName(sbName)
			sb.SetNamespace(gardenerKymaNamespace)
			sb.SetLabels(labels)
			sb.SetSecretRefName(sbName)
			sb.SetSecretRefNamespace(gardenerKymaNamespace)
			unstructuredObj = &sb.Unstructured
			_, err := gardenerClient.Resource(gardener.SecretBindingResource).Namespace(gardenerKymaNamespace).Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
			require.NoError(t, err)
		} else {
			cb := &gardener.CredentialsBinding{}
			cb.SetName(sbName)
			cb.SetNamespace(gardenerKymaNamespace)
			cb.SetLabels(labels)
			cb.SetSecretRefName(sbName)
			cb.SetSecretRefNamespace(gardenerKymaNamespace)
			unstructuredObj = &cb.Unstructured
			_, err := gardenerClient.Resource(gardener.CredentialsBindingResource).Namespace(gardenerKymaNamespace).Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
			require.NoError(t, err)
		}

	}
}

func defaultOIDCValues() pkg.OIDCConfigDTO {
	return pkg.OIDCConfigDTO{
		ClientID:       "client-id-oidc",
		GroupsClaim:    "groups",
		GroupsPrefix:   "-",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func workersProvider(imConfig broker.InfrastructureManager, spec *configuration.ProviderSpec) *workers.Provider {
	return workers.NewProvider(
		imConfig,
		spec,
	)
}

func (s *BrokerSuiteTest) SetRuntimeResourceStateReady(runtimeID string) {
	s.processInfrastructureManagerProvisioningRuntimeResource(runtimeID, "Ready")
}

func (s *BrokerSuiteTest) SetRuntimeResourceFailed(runtimeID string) {
	s.processInfrastructureManagerProvisioningRuntimeResource(runtimeID, "Failed")
}

func (s *BrokerSuiteTest) processInfrastructureManagerProvisioningRuntimeResource(runtimeID string, state string) {
	err := s.poller.Invoke(func() (bool, error) {
		runtimeResource := &unstructured.Unstructured{}
		gvk, _ := customresources.GvkByName(customresources.RuntimeCr)
		runtimeResource.SetGroupVersionKind(gvk)
		err := s.k8sKcp.Get(context.Background(), client.ObjectKey{
			Namespace: "kyma-system",
			Name:      steps.KymaRuntimeResourceNameFromID(runtimeID),
		}, runtimeResource)
		if err != nil {
			return false, nil
		}

		err = unstructured.SetNestedField(runtimeResource.Object, state, "status", "state")
		assert.NoError(s.t, err)
		err = s.k8sKcp.Update(context.Background(), runtimeResource)
		return err == nil, nil
	})
	assert.NoError(s.t, err)
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

func (s *BrokerSuiteTest) CreateAPI(cfg *Config, db storage.BrokerStorage, provisioningQueue *process.Queue,
	deprovisionQueue *process.Queue, updateQueue *process.Queue, log *slog.Logger,
	skrK8sClientProvider *kubeconfig.FakeProvider, eventBroker *event.PubSub, configProvider kebConfig.Provider, planSpec *configuration.PlanSpecifications,
	rulesService *rules.RulesService, gardenerClient *gardener.Client, awsClientFactory aws.ClientFactory) {
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
				broker.BuildRuntimeAWSPlanName: {
					Description: broker.BuildRuntimeAWSPlanName,
					Metadata:    broker.PlanMetadata{},
				},
				broker.BuildRuntimeGCPPlanID: {
					Description: broker.BuildRuntimeGCPPlanName,
					Metadata:    broker.PlanMetadata{},
				},
				broker.BuildRuntimeAzurePlanName: {
					Description: broker.BuildRuntimeAzurePlanName,
					Metadata:    broker.PlanMetadata{},
				},
				broker.AlicloudPlanID: {
					Description: broker.AlicloudPlanName,
					Metadata:    broker.PlanMetadata{},
				},
			},
		},
	}
	var fakeKcpK8sClient = fake.NewClientBuilder().Build()
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("Build", nil).Return("--kubeconfig file", nil)
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://api.server.url.dummy", nil)

	providerSpec, err := configuration.NewProviderSpecFromFile(cfg.ProvidersConfigurationFilePath)
	fatalOnError(err, log)

	defaultOIDC := defaultOIDCValues()
	runtimeConfigProvider := kebConfig.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, kebConfig.RuntimeConfigurationRequiredFields)
	channelResolver, err := kebConfig.NewChannelResolver(runtimeConfigProvider, broker.AllPlanNames(), log)
	fatalOnError(err, log)
	schemaService := broker.NewSchemaService(providerSpec, planSpec, &defaultOIDC, cfg.Broker, cfg.InfrastructureManager.IngressFilteringPlans, channelResolver)

	createAPI(s.router, schemaService, servicesConfig, cfg, db, provisioningQueue, deprovisionQueue, updateQueue,
		lager.NewLogger("api"), log, kcBuilder, skrK8sClientProvider, skrK8sClientProvider, fakeKcpK8sClient, eventBroker, defaultOIDCValues(),
		providerSpec, configProvider, planSpec, rulesService, gardenerClient, awsClientFactory)

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
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
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

func (s *BrokerSuiteTest) WaitForInstanceRemoval(iid string) {
	err := s.poller.Invoke(func() (done bool, err error) {
		_, err = s.db.Instances().GetByID(iid)
		return dberr.IsNotFound(err), nil
	})
	assert.NoError(s.t, err, "timeout waiting for the instance %s to be removed", iid)
}

func (s *BrokerSuiteTest) AssertBindingRemoval(iid string, bindingID string) {
	_, err := s.db.Bindings().Get(iid, bindingID)
	assert.True(s.t, dberr.IsNotFound(err), "bindings should be removed")
}

func (s *BrokerSuiteTest) LastOperation(iid string) *internal.Operation {
	op, _ := s.db.Operations().GetLastOperation(iid)
	return op
}

func (s *BrokerSuiteTest) FinishProvisioningOperationByInfrastructureManager(operationID string) {
	var op *internal.ProvisioningOperation
	err := s.poller.Invoke(func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.SetRuntimeResourceStateReady(op.RuntimeID)
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

func (s *BrokerSuiteTest) Log(msg string) {
	s.t.Helper()
	s.t.Log(msg)
}

func (s *BrokerSuiteTest) GetInstance(iid string) *internal.Instance {
	inst, err := s.db.Instances().GetByID(iid)
	require.NoError(s.t, err)
	return inst
}

func (s *BrokerSuiteTest) processKIMProvisioningByOperationID(opID string) {
	s.WaitForProvisioningState(opID, domain.InProgress)

	s.FinishProvisioningOperationByInfrastructureManager(opID)
}

func (s *BrokerSuiteTest) processKIMProvisioningByInstanceID(iid string) {
	var runtimeID string
	err := s.poller.Invoke(func() (done bool, err error) {
		instance, _ := s.db.Instances().GetByID(iid)
		if instance.RuntimeID != "" {
			runtimeID = instance.RuntimeID
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the runtimeID to be created.")

	s.SetRuntimeResourceStateReady(runtimeID)
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

	err := s.poller.Invoke(func() (done bool, err error) {
		err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertRuntimeResourceExists(opId string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)

	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	gvk, err := customresources.GvkByName(customresources.RuntimeCr)
	assert.NoError(s.t, err)
	obj.SetGroupVersionKind(gvk)

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertRuntimeResourceWorkers(obj *unstructured.Unstructured, machineType string, autoscalerMax, autoscalerMin int64) {
	workers, found, err := unstructured.NestedSlice(obj.Object, "spec", "shoot", "provider", "workers")
	assert.NoError(s.t, err)
	assert.True(s.t, found, "Workers should be present in the runtime resource spec")
	actualMachineType, found, err := unstructured.NestedString(workers[0].(map[string]interface{}), "machine", "type")
	assert.NoError(s.t, err)
	assert.True(s.t, found, "Machine type should be present in the worker spec")
	assert.Equal(s.t, machineType, actualMachineType, "Machine type in the runtime resource spec should match the expected value")
	actualMax, found, err := unstructured.NestedInt64(workers[0].(map[string]interface{}), "maximum")
	assert.NoError(s.t, err)
	assert.True(s.t, found, "Autoscaler maximum should be present in the worker spec")
	assert.Equal(s.t, autoscalerMax, actualMax, "Autoscaler maximum in the runtime resource spec should match the expected value")
	actualMin, found, err := unstructured.NestedInt64(workers[0].(map[string]interface{}), "minimum")
	assert.NoError(s.t, err)
	assert.True(s.t, found, "Autoscaler minimum should be present in the worker spec")
	assert.Equal(s.t, autoscalerMin, actualMin, "Autoscaler minimum in the runtime resource spec should match the expected value")
}

func (s *BrokerSuiteTest) GetUnstructuredRuntimeResource(opId string) *unstructured.Unstructured {

	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)
	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	gvk, err := customresources.GvkByName(customresources.RuntimeCr)
	assert.NoError(s.t, err)
	obj.SetGroupVersionKind(gvk)

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
	assert.NoError(s.t, err)

	return obj
}

func (s *BrokerSuiteTest) AssertRuntimeResourceLabels(opId string) {
	operation, err := s.db.Operations().GetOperationByID(opId)
	assert.NoError(s.t, err)
	obj := &unstructured.Unstructured{}
	obj.SetName(operation.RuntimeID)
	obj.SetNamespace("kyma-system")
	gvk, err := customresources.GvkByName(customresources.RuntimeCr)
	assert.NoError(s.t, err)
	obj.SetGroupVersionKind(gvk)

	err = s.k8sKcp.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
	assert.NoError(s.t, err)

	assert.Subset(s.t, obj.GetLabels(), map[string]string{customresources.InstanceIdLabel: operation.InstanceID,
		customresources.GlobalAccountIdLabel: operation.ProvisioningParameters.ErsContext.GlobalAccountID,
		customresources.SubaccountIdLabel:    operation.ProvisioningParameters.ErsContext.SubAccountID,
		customresources.ShootNameLabel:       operation.ShootName,
		customresources.PlanIdLabel:          operation.ProvisioningParameters.PlanID,
		customresources.RuntimeIdLabel:       operation.RuntimeID,
		customresources.PlatformRegionLabel:  operation.ProvisioningParameters.PlatformRegion,
		customresources.RegionLabel:          operation.Region,
		customresources.CloudProviderLabel:   operation.CloudProvider,
		customresources.PlanNameLabel:        broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
	})
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
	a := s.metrics.OperationResult.Metrics().With(metrics.GetLabels(operation))
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

func (s *BrokerSuiteTest) AssertRuntimeAdminsByInstanceID(id string, admins []string) {
	runtime := s.GetRuntimeResourceByInstanceID(id)
	assert.Equal(s.t, admins, runtime.Spec.Security.Administrators)
}

func (s *BrokerSuiteTest) AssertNetworkFiltering(iid string, egressExpected bool, ingressExpected bool) {
	runtime := s.GetRuntimeResourceByInstanceID(iid)
	assert.Equal(s.t, egressExpected, runtime.Spec.Security.Networking.Filter.Egress.Enabled)
	assert.Equal(s.t, ingressExpected, runtime.Spec.Security.Networking.Filter.Ingress.Enabled)
}

func (s *BrokerSuiteTest) failRuntimeByKIM(iid string) {
	err := s.poller.Invoke(func() (bool, error) {
		var runtimes imv1.RuntimeList
		err := s.k8sKcp.List(context.Background(), &runtimes, client.MatchingLabels{"kyma-project.io/instance-id": iid})
		require.NoError(s.t, err)
		if len(runtimes.Items) == 0 {
			return false, nil
		}

		runtime := runtimes.Items[0]
		runtime.Status.State = imv1.RuntimeStateFailed

		err = s.k8sKcp.Update(context.Background(), &runtime)
		return true, nil
	})
	require.NoError(s.t, err)
}

func (s *BrokerSuiteTest) FinishDeprovisioningOperationByKIM(opID string) {
	op, err := s.db.Operations().GetOperationByID(opID)
	require.NoError(s.t, err)
	iid := op.InstanceID
	runtime := s.GetRuntimeResourceByInstanceID(iid)
	s.k8sDeletionObjectTracker.ProcessRuntimeDeletion(runtime.Name)
}

func (s *BrokerSuiteTest) AssertBTPOperatorSecret() {
	secret := &corev1.Secret{}
	err := s.k8sSKR.Get(context.Background(), client.ObjectKey{Namespace: "kyma-installer", Name: "btp-operator"}, secret)
	require.NoError(s.t, err)
	assert.Equal(s.t, "btp-operator", secret.Name)
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

func (s *BrokerSuiteTest) assertAdditionalWorkerIsCreated(t *testing.T, provider imv1.Provider, name, machineType string, autoScalerMin, autoScalerMax, zonesNumer int) {
	var worker *v1beta1.Worker
	for _, additionalWorker := range *provider.AdditionalWorkers {
		if additionalWorker.Name == name {
			worker = &additionalWorker
		}
	}
	require.NotNil(t, worker)
	assert.Equal(t, machineType, worker.Machine.Type)
	assert.Equal(t, int32(autoScalerMin), worker.Minimum)
	assert.Equal(t, int32(autoScalerMax), worker.Maximum)
	assert.Equal(t, zonesNumer, worker.MaxSurge.IntValue())
	assert.Len(t, worker.Zones, zonesNumer)
}

func (s *BrokerSuiteTest) assertAdditionalWorkerZones(t *testing.T, provider imv1.Provider, name string, zonesNumber int, zones ...string) {
	var worker *v1beta1.Worker
	for _, additionalWorker := range *provider.AdditionalWorkers {
		if additionalWorker.Name == name {
			worker = &additionalWorker
		}
	}
	require.NotNil(t, worker)
	assert.Len(t, worker.Zones, zonesNumber)
	for _, v := range worker.Zones {
		assert.Contains(t, zones, v)
	}
}

func fixSecrets() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sb-aws",
				Namespace: "kyma",
			},
			Data: map[string][]byte{
				"accessKeyID":     []byte("test-key"),
				"secretAccessKey": []byte("test-secret"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sb-aws-shared",
				Namespace: "kyma",
			},
			Data: map[string][]byte{
				"accessKeyID":     []byte("test-key"),
				"secretAccessKey": []byte("test-secret"),
			},
		},
	}
}

func fixDiscoveredZones() map[string][]string {
	return map[string][]string{
		"m6i.large": {"zone-d", "zone-e", "zone-f", "zone-g"},
		"m5.xlarge": {"zone-h", "zone-i", "zone-j", "zone-k"},
		"c7i.large": {"zone-l", "zone-m"},
	}
}
