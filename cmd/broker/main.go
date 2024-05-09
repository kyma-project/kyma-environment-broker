package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	gruntime "runtime"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	orchestrationExt "github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	eventshandler "github.com/kyma-project/kyma-environment-broker/internal/events/handler"
	"github.com/kyma-project/kyma-environment-broker/internal/health"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/kyma-environment-broker/internal/runtimeoverrides"
	"github.com/kyma-project/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/suspension"
	"github.com/kyma-project/kyma-environment-broker/internal/swagger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Config holds configuration for the whole application
type Config struct {
	// DbInMemory allows to use memory storage instead of the postgres one.
	// Suitable for development purposes.
	DbInMemory bool `envconfig:"default=false"`

	// DisableProcessOperationsInProgress allows to disable processing operations
	// which are in progress on starting application. Set to true if you are
	// running in a separate testing deployment but with the production DB.
	DisableProcessOperationsInProgress bool `envconfig:"default=false"`

	// DevelopmentMode if set to true then errors are returned in http
	// responses, otherwise errors are only logged and generic message
	// is returned to client.
	// Currently works only with /info endpoints.
	DevelopmentMode bool `envconfig:"default=false"`

	// DumpProvisionerRequests enables dumping Provisioner requests. Must be disabled on Production environments
	// because some data must not be visible in the log file.
	DumpProvisionerRequests bool `envconfig:"default=false"`

	// OperationTimeout is used to check on a top-level if any operation didn't exceed the time for processing.
	// It is used for provisioning and deprovisioning operations.
	OperationTimeout time.Duration `envconfig:"default=24h"`

	Host       string `envconfig:"optional"`
	Port       string `envconfig:"default=8080"`
	StatusPort string `envconfig:"default=8071"`

	Provisioner input.Config
	Reconciler  reconciler.Config
	Database    storage.Config
	Gardener    gardener.Config
	Kubeconfig  kubeconfig.Config

	KymaVersion                                                         string
	EnableOnDemandVersion                                               bool `envconfig:"default=false"`
	ManagedRuntimeComponentsYAMLFilePath                                string
	NewAdditionalRuntimeComponentsYAMLFilePath                          string
	SkrOidcDefaultValuesYAMLFilePath                                    string
	SkrDnsProvidersValuesYAMLFilePath                                   string
	DefaultRequestRegion                                                string `envconfig:"default=cf-eu10"`
	UpdateProcessingEnabled                                             bool   `envconfig:"default=false"`
	UpdateSubAccountMovementEnabled                                     bool   `envconfig:"default=false"`
	LifecycleManagerIntegrationDisabled                                 bool   `envconfig:"default=true"`
	ReconcilerIntegrationDisabled                                       bool   `envconfig:"default=false"`
	InfrastructureManagerIntegrationDisabled                            bool   `envconfig:"default=true"`
	AvsMaintenanceModeDuringUpgradeAlwaysDisabledGlobalAccountsFilePath string
	Broker                                                              broker.Config
	CatalogFilePath                                                     string

	Avs avs.Config
	IAS ias.Config
	EDP edp.Config

	Notification notification.Config

	VersionConfig struct {
		Namespace string
		Name      string
	}

	KymaDashboardConfig dashboard.Config

	OrchestrationConfig orchestration.Config

	TrialRegionMappingFilePath string

	EuAccessWhitelistedGlobalAccountsFilePath string
	EuAccessRejectionMessage                  string `envconfig:"default=Due to limited availability you need to open support ticket before attempting to provision Kyma clusters in EU Access only regions"`

	MaxPaginationPage int `envconfig:"default=100"`

	LogLevel string `envconfig:"default=info"`

	// FreemiumProviders is a list of providers for freemium
	FreemiumProviders []string `envconfig:"default=aws"`

	FreemiumWhitelistedGlobalAccountsFilePath string

	DomainName string

	// Enable/disable profiler configuration. The profiler samples will be stored
	// under /tmp/profiler directory. Based on the deployment strategy, this will be
	// either ephemeral container filesystem or persistent storage
	Profiler ProfilerConfig

	Events events.Config

	MetricsV2 metricsv2.Config

	Provisioning    process.StagedManagerConfiguration
	Deprovisioning  process.StagedManagerConfiguration
	Update          process.StagedManagerConfiguration
	ArchiveEnabled  bool `envconfig:"default=false"`
	ArchiveDryRun   bool `envconfig:"default=true"`
	CleaningEnabled bool `envconfig:"default=false"`
	CleaningDryRun  bool `envconfig:"default=true"`

	KymaResourceDeletionTimeout time.Duration `envconfig:"default=30s"`
}

type ProfilerConfig struct {
	Path     string        `envconfig:"default=/tmp/profiler"`
	Sampling time.Duration `envconfig:"default=1s"`
	Memory   bool
}

type K8sClientProvider interface {
	K8sClientForRuntimeID(rid string) (client.Client, error)
}

type KubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeId string) ([]byte, error)
}

const (
	createRuntimeStageName      = "create_runtime"
	checkKymaStageName          = "check_kyma"
	createKymaResourceStageName = "create_kyma_resource"
	startStageName              = "start"
)

func periodicProfile(logger lager.Logger, profiler ProfilerConfig) {
	if profiler.Memory == false {
		return
	}
	logger.Info(fmt.Sprintf("Starting periodic profiler %v", profiler))
	if err := os.MkdirAll(profiler.Path, os.ModePerm); err != nil {
		logger.Error(fmt.Sprintf("Failed to create dir %v for profile storage", profiler.Path), err)
	}
	for {
		profName := fmt.Sprintf("%v/mem-%v.pprof", profiler.Path, time.Now().Unix())
		logger.Info(fmt.Sprintf("Creating periodic memory profile %v", profName))
		profFile, err := os.Create(profName)
		if err != nil {
			logger.Error(fmt.Sprintf("Creating periodic memory profile %v failed", profName), err)
		}
		err = pprof.Lookup("allocs").WriteTo(profFile, 0)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to write periodic memory profile to %v file", profName), err)
		}
		gruntime.GC()
		time.Sleep(profiler.Sampling)
	}
}

func main() {
	err := apiextensionsv1.AddToScheme(scheme.Scheme)
	panicOnError(err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set default formatted
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	// create and fill config
	var cfg Config
	err = envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err, logs)

	if cfg.LogLevel != "" {
		l, _ := logrus.ParseLevel(cfg.LogLevel)
		logs.SetLevel(l)
	}

	// check default Kyma versions
	err = checkDefaultVersions(cfg.KymaVersion)
	panicOnError(err)

	cfg.OrchestrationConfig.KymaVersion = cfg.KymaVersion
	cfg.OrchestrationConfig.KubernetesVersion = cfg.Provisioner.KubernetesVersion

	// create logger
	logger := lager.NewLogger("kyma-env-broker")

	logger.Info("Starting Kyma Environment Broker")

	logger.Info("Registering healthz endpoint for health probes")
	health.NewServer(cfg.Host, cfg.StatusPort, logs).ServeAsync()
	go periodicProfile(logger, cfg.Profiler)

	logs.Infof("Setting provisioner timeouts: provisioning=%s, deprovisioning=%s", cfg.Provisioner.ProvisioningTimeout, cfg.Provisioner.DeprovisioningTimeout)
	logs.Infof("Setting reconciler timeout: provisioning=%s", cfg.Reconciler.ProvisioningTimeout)
	logs.Infof("Setting staged manager configuration: provisioning=%s, deprovisioning=%s, update=%s", cfg.Provisioning, cfg.Deprovisioning, cfg.Update)
	logs.Infof("InfrastructureManagerIntegrationDisabled: %v", cfg.InfrastructureManagerIntegrationDisabled)
	logs.Infof("Archiving enabled: %v, dry run: %v", cfg.ArchiveEnabled, cfg.ArchiveDryRun)
	logs.Infof("Cleaning enabled: %v, dry run: %v", cfg.CleaningEnabled, cfg.CleaningDryRun)

	// create provisioner client
	provisionerClient := provisioner.NewProvisionerClient(cfg.Provisioner.URL, cfg.DumpProvisionerRequests, logs.WithField("service", "provisioner"))

	reconcilerClient := reconciler.NewReconcilerClient(http.DefaultClient, logs.WithField("service", "reconciler"), &cfg.Reconciler)

	// create kubernetes client
	k8sCfg, err := config.GetConfig()
	fatalOnError(err, logs)
	cli, err := initClient(k8sCfg)
	fatalOnError(err, logs)
	skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(cli)

	// create storage
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	var db storage.BrokerStorage
	if cfg.DbInMemory {
		db = storage.NewMemoryStorage()
	} else {
		store, conn, err := storage.NewFromConfig(cfg.Database, cfg.Events, cipher, logs.WithField("service", "storage"))
		fatalOnError(err, logs)
		db = store
		dbStatsCollector := sqlstats.NewStatsCollector("broker", conn)
		prometheus.MustRegister(dbStatsCollector)
	}

	// Customer Notification
	clientHTTPForNotification := httputil.NewClient(60, true)
	notificationClient := notification.NewClient(clientHTTPForNotification, notification.ClientConfig{
		URL: cfg.Notification.Url,
	})
	notificationBuilder := notification.NewBundleBuilder(notificationClient, cfg.Notification)

	// Register disabler. Convention:
	// {component-name} : {component-disabler-service}
	//
	// Using map is intentional - we ensure that component name is not duplicated.
	optionalComponentsDisablers := runtime.ComponentsDisablers{
		components.Kiali:   runtime.NewGenericComponentDisabler(components.Kiali),
		components.Tracing: runtime.NewGenericComponentDisabler(components.Tracing),
	}
	optComponentsSvc := runtime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := runtime.NewDisabledComponentsProvider()

	// provides configuration for specified Kyma version and plan
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logs, cfg.KymaVersion),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	componentsProvider := runtime.NewComponentsProvider()
	gardenerClusterConfig, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	fatalOnError(err, logs)
	cfg.Gardener.DNSProviders, err = gardener.ReadDNSProvidersValuesFromYAML(cfg.SkrDnsProvidersValuesYAMLFilePath)
	fatalOnError(err, logs)
	dynamicGardener, err := dynamic.NewForConfig(gardenerClusterConfig)
	fatalOnError(err, logs)

	gardenerNamespace := fmt.Sprintf("garden-%v", cfg.Gardener.Project)
	gardenerAccountPool := hyperscaler.NewAccountPool(dynamicGardener, gardenerNamespace)
	gardenerSharedPool := hyperscaler.NewSharedGardenerAccountPool(dynamicGardener, gardenerNamespace)
	accountProvider := hyperscaler.NewAccountProvider(gardenerAccountPool, gardenerSharedPool)

	regions, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	fatalOnError(err, logs)
	logs.Infof("Platform region mapping for trial: %v", regions)

	oidcDefaultValues, err := runtime.ReadOIDCDefaultValuesFromYAML(cfg.SkrOidcDefaultValuesYAMLFilePath)
	fatalOnError(err, logs)
	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentsProvider,
		configProvider, cfg.Provisioner, cfg.KymaVersion, regions, cfg.FreemiumProviders, oidcDefaultValues)
	fatalOnError(err, logs)

	edpClient := edp.NewClient(cfg.EDP, logs.WithField("service", "edpClient"))

	panicOnError(cfg.Avs.ReadMaintenanceModeDuringUpgradeAlwaysDisabledGAIDsFromYaml(
		cfg.AvsMaintenanceModeDuringUpgradeAlwaysDisabledGlobalAccountsFilePath))
	avsClient, err := avs.NewClient(ctx, cfg.Avs, logs)
	fatalOnError(err, logs)
	avsDel := avs.NewDelegator(avsClient, cfg.Avs, db.Operations())
	externalEvalAssistant := avs.NewExternalEvalAssistant(cfg.Avs)
	internalEvalAssistant := avs.NewInternalEvalAssistant(cfg.Avs)
	externalEvalCreator := provisioning.NewExternalEvalCreator(avsDel, cfg.Avs.ExternalTesterDisabled, externalEvalAssistant)
	upgradeEvalManager := avs.NewEvaluationManager(avsDel, cfg.Avs)

	// IAS
	clientHTTPForIAS := httputil.NewClient(60, cfg.IAS.SkipCertVerification)
	if cfg.IAS.TLSRenegotiationEnable {
		clientHTTPForIAS = httputil.NewRenegotiationTLSClient(30, cfg.IAS.SkipCertVerification)
	}
	iasClient := ias.NewClient(clientHTTPForIAS, ias.ClientConfig{
		URL:    cfg.IAS.URL,
		ID:     cfg.IAS.UserID,
		Secret: cfg.IAS.UserSecret,
	})
	bundleBuilder := ias.NewBundleBuilder(iasClient, cfg.IAS)

	// application event broker
	eventBroker := event.NewPubSub(logs)

	// metrics collectors
	metrics.Register(ctx, eventBroker, db.Operations(), db.Instances(), logs)
	if cfg.MetricsV2.Enabled {
		_ = metricsv2.Register(ctx, eventBroker, db.Operations(), db.Instances(), cfg.MetricsV2, logs)
	}

	// setup runtime overrides appender
	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)

	// define steps
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, accountVersionMapping, db.RuntimeStates())

	// run queues
	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Provisioning, logs.WithField("provisioning", "manager"))
	provisionQueue := NewProvisioningProcessingQueue(ctx, provisionManager, cfg.Provisioning.WorkersAmount, &cfg, db, provisionerClient, inputFactory,
		avsDel, internalEvalAssistant, externalEvalCreator, runtimeVerConfigurator,
		runtimeOverrides, edpClient, accountProvider, reconcilerClient, skrK8sClientProvider, cli, logs)

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Deprovisioning, logs.WithField("deprovisioning", "manager"))
	deprovisionQueue := NewDeprovisioningProcessingQueue(ctx, cfg.Deprovisioning.WorkersAmount, deprovisionManager, &cfg, db, eventBroker, provisionerClient,
		avsDel, internalEvalAssistant, externalEvalAssistant, bundleBuilder, edpClient, accountProvider, reconcilerClient,
		skrK8sClientProvider, cli, configProvider, logs)

	updateManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Update, logs.WithField("update", "manager"))
	updateQueue := NewUpdateProcessingQueue(ctx, updateManager, cfg.Update.WorkersAmount, db, inputFactory, provisionerClient, eventBroker,
		runtimeVerConfigurator, db.RuntimeStates(), componentsProvider, reconcilerClient, cfg, skrK8sClientProvider, cli, logs)
	/***/
	servicesConfig, err := broker.NewServicesConfigFromFile(cfg.CatalogFilePath)
	fatalOnError(err, logs)

	// create server
	router := mux.NewRouter()

	createAPI(router, servicesConfig, inputFactory, &cfg, db, provisionQueue, deprovisionQueue, updateQueue, logger, logs, inputFactory.GetPlanDefaults)

	// create metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// create SKR kubeconfig endpoint
	kcBuilder := kubeconfig.NewBuilder(provisionerClient, skrK8sClientProvider)
	kcHandler := kubeconfig.NewHandler(db, kcBuilder, cfg.Kubeconfig.AllowOrigins, logs.WithField("service", "kubeconfigHandle"))
	kcHandler.AttachRoutes(router)

	runtimeLister := orchestration.NewRuntimeLister(db.Instances(), db.Operations(), runtime.NewConverter(cfg.DefaultRequestRegion), logs)
	runtimeResolver := orchestrationExt.NewGardenerRuntimeResolver(dynamicGardener, gardenerNamespace, runtimeLister, logs)

	kymaQueue := NewKymaOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, eventBroker, inputFactory, nil, time.Minute, runtimeVerConfigurator, runtimeResolver, upgradeEvalManager, &cfg, internalEvalAssistant, reconcilerClient, notificationBuilder, skrK8sClientProvider, logs, cli, 1)
	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory,
		nil, time.Minute, runtimeResolver, upgradeEvalManager, notificationBuilder, logs, cli, cfg, 1)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, kymaQueue, clusterQueue, cfg.MaxPaginationPage, logs)

	if !cfg.DisableProcessOperationsInProgress {
		err = processOperationsInProgressByType(internal.OperationTypeProvision, db.Operations(), provisionQueue, logs)
		fatalOnError(err, logs)
		err = processOperationsInProgressByType(internal.OperationTypeDeprovision, db.Operations(), deprovisionQueue, logs)
		fatalOnError(err, logs)
		err = processOperationsInProgressByType(internal.OperationTypeUpdate, db.Operations(), updateQueue, logs)
		fatalOnError(err, logs)
		err = reprocessOrchestrations(orchestrationExt.UpgradeKymaOrchestration, db.Orchestrations(), db.Operations(), kymaQueue, logs)
		fatalOnError(err, logs)
		err = reprocessOrchestrations(orchestrationExt.UpgradeClusterOrchestration, db.Orchestrations(), db.Operations(), clusterQueue, logs)
		fatalOnError(err, logs)
	} else {
		logger.Info("Skipping processing operation in progress on start")
	}

	// configure templates e.g. {{.domain}} to replace it with the domain name
	swaggerTemplates := map[string]string{
		"domain": cfg.DomainName,
	}
	err = swagger.NewTemplate("/swagger", swaggerTemplates).Execute()
	fatalOnError(err, logs)

	// create /orchestration
	orchestrationHandler.AttachRoutes(router)

	// create list runtimes endpoint
	runtimeHandler := runtime.NewHandler(db.Instances(), db.Operations(),
		db.RuntimeStates(), db.InstancesArchived(), cfg.MaxPaginationPage,
		cfg.DefaultRequestRegion, provisionerClient, logs)
	runtimeHandler.AttachRoutes(router)

	// create expiration endpoint
	expirationHandler := expiration.NewHandler(db.Instances(), db.Operations(), deprovisionQueue, logs)
	expirationHandler.AttachRoutes(router)

	router.StrictSlash(true).PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("/swagger"))))
	svr := handlers.CustomLoggingHandler(os.Stdout, router, func(writer io.Writer, params handlers.LogFormatterParams) {
		logs.Infof("Call handled: method=%s url=%s statusCode=%d size=%d", params.Request.Method, params.URL.Path, params.StatusCode, params.Size)
	})

	fatalOnError(http.ListenAndServe(cfg.Host+":"+cfg.Port, svr), logs)
}

func checkDefaultVersions(versions ...string) error {
	for _, version := range versions {
		if !isVersionFollowingSemanticVersioning(version) {
			return fmt.Errorf("Kyma default versions are not following semantic versioning")
		}
	}
	return nil
}

func isVersionFollowingSemanticVersioning(version string) bool {
	regexpToMatch := regexp.MustCompile("(^[0-9]+\\.{1}).*")
	if regexpToMatch.MatchString(version) {
		return true
	}
	return false
}

func createAPI(router *mux.Router, servicesConfig broker.ServicesConfig, planValidator broker.PlanValidator, cfg *Config, db storage.BrokerStorage, provisionQueue, deprovisionQueue, updateQueue *process.Queue, logger lager.Logger, logs logrus.FieldLogger, planDefaults broker.PlanDefaults) {
	suspensionCtxHandler := suspension.NewContextUpdateHandler(db.Operations(), provisionQueue, deprovisionQueue, logs)

	defaultPlansConfig, err := servicesConfig.DefaultPlansConfig()
	fatalOnError(err, logs)

	debugSink, err := lager.NewRedactingSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), []string{"instance-details"}, []string{})
	fatalOnError(err, logs)
	logger.RegisterSink(debugSink)
	errorSink, err := lager.NewRedactingSink(lager.NewWriterSink(os.Stderr, lager.ERROR), []string{"instance-details"}, []string{})
	fatalOnError(err, logs)
	logger.RegisterSink(errorSink)

	// EU Access whitelisting
	whitelistedGlobalAccountIds, err := whitelist.ReadWhitelistedGlobalAccountIdsFromFile(cfg.EuAccessWhitelistedGlobalAccountsFilePath)
	fatalOnError(err, logs)
	logs.Infof("Number of globalAccountIds for EU Access: %d\n", len(whitelistedGlobalAccountIds))

	freemiumGlobalAccountIds, err := whitelist.ReadWhitelistedGlobalAccountIdsFromFile(cfg.FreemiumWhitelistedGlobalAccountsFilePath)
	fatalOnError(err, logs)
	logs.Infof("Number of globalAccountIds for unlimited freeemium: %d\n", len(freemiumGlobalAccountIds))

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		ServicesEndpoint: broker.NewServices(cfg.Broker, servicesConfig, logs),
		ProvisionEndpoint: broker.NewProvision(cfg.Broker, cfg.Gardener, db.Operations(), db.Instances(), db.InstancesArchived(),
			provisionQueue, planValidator, defaultPlansConfig, cfg.EnableOnDemandVersion,
			planDefaults, whitelistedGlobalAccountIds, cfg.EuAccessRejectionMessage, logs, cfg.KymaDashboardConfig, freemiumGlobalAccountIds),
		DeprovisionEndpoint: broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		UpdateEndpoint: broker.NewUpdate(cfg.Broker, db.Instances(), db.RuntimeStates(), db.Operations(),
			suspensionCtxHandler, cfg.UpdateProcessingEnabled, cfg.UpdateSubAccountMovementEnabled, updateQueue, defaultPlansConfig,
			planDefaults, logs, cfg.KymaDashboardConfig),
		GetInstanceEndpoint:          broker.NewGetInstance(cfg.Broker, db.Instances(), db.Operations(), logs),
		LastOperationEndpoint:        broker.NewLastOperation(db.Operations(), db.InstancesArchived(), logs),
		BindEndpoint:                 broker.NewBind(cfg.Broker.Binding, db.Instances(), logs),
		UnbindEndpoint:               broker.NewUnbind(logs),
		GetBindingEndpoint:           broker.NewGetBinding(logs),
		LastBindingOperationEndpoint: broker.NewLastBindingOperation(logs),
	}

	router.Use(middleware.AddRegionToContext(cfg.DefaultRequestRegion))
	router.Use(middleware.AddProviderToContext())
	for _, prefix := range []string{
		"/oauth/",          // oauth2 handled by Ory
		"/oauth/{region}/", // oauth2 handled by Ory with region
	} {
		route := router.PathPrefix(prefix).Subrouter()
		broker.AttachRoutes(route, kymaEnvBroker, logger)
	}

	respWriter := httputil.NewResponseWriter(logs, cfg.DevelopmentMode)
	runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(db.Instances(), db.Operations(), defaultPlansConfig, cfg.DefaultRequestRegion, respWriter)
	router.Handle("/info/runtimes", runtimesInfoHandler)
	router.Handle("/events", eventshandler.NewHandler(db.Events(), db.Instances()))
}

// queues all in progress operations by type
func processOperationsInProgressByType(opType internal.OperationType, op storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	operations, err := op.GetNotFinishedOperationsByType(opType)
	if err != nil {
		return fmt.Errorf("while getting in progress operations from storage: %w", err)
	}
	for _, operation := range operations {
		queue.Add(operation.ID)
		log.Infof("Resuming the processing of %s operation ID: %s", opType, operation.ID)
	}
	return nil
}

func reprocessOrchestrations(orchestrationType orchestrationExt.Type, orchestrationsStorage storage.Orchestrations, operationsStorage storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	if err := processCancelingOrchestrations(orchestrationType, orchestrationsStorage, operationsStorage, queue, log); err != nil {
		return fmt.Errorf("while processing canceled %s orchestrations: %w", orchestrationType, err)
	}
	if err := processOrchestration(orchestrationType, orchestrationExt.InProgress, orchestrationsStorage, queue, log); err != nil {
		return fmt.Errorf("while processing in progress %s orchestrations: %w", orchestrationType, err)
	}
	if err := processOrchestration(orchestrationType, orchestrationExt.Pending, orchestrationsStorage, queue, log); err != nil {
		return fmt.Errorf("while processing pending %s orchestrations: %w", orchestrationType, err)
	}
	if err := processOrchestration(orchestrationType, orchestrationExt.Retrying, orchestrationsStorage, queue, log); err != nil {
		return fmt.Errorf("while processing retrying %s orchestrations: %w", orchestrationType, err)
	}
	return nil
}

func processOrchestration(orchestrationType orchestrationExt.Type, state string, orchestrationsStorage storage.Orchestrations, queue *process.Queue, log logrus.FieldLogger) error {
	filter := dbmodel.OrchestrationFilter{
		Types:  []string{string(orchestrationType)},
		States: []string{state},
	}
	orchestrations, _, _, err := orchestrationsStorage.List(filter)
	if err != nil {
		return fmt.Errorf("while getting %s %s orchestrations from storage: %w", state, orchestrationType, err)
	}
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})

	for _, o := range orchestrations {
		queue.Add(o.OrchestrationID)
		log.Infof("Resuming the processing of %s %s orchestration ID: %s", state, orchestrationType, o.OrchestrationID)
	}
	return nil
}

// processCancelingOrchestrations reprocess orchestrations with canceling state only when some in progress operations exists
// reprocess only one orchestration to not clog up the orchestration queue on start
func processCancelingOrchestrations(orchestrationType orchestrationExt.Type, orchestrationsStorage storage.Orchestrations, operationsStorage storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	filter := dbmodel.OrchestrationFilter{
		Types:  []string{string(orchestrationType)},
		States: []string{orchestrationExt.Canceling},
	}
	orchestrations, _, _, err := orchestrationsStorage.List(filter)
	if err != nil {
		return fmt.Errorf("while getting canceling %s orchestrations from storage: %w", orchestrationType, err)
	}
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})

	for _, o := range orchestrations {
		count := 0
		err = nil
		if orchestrationType == orchestrationExt.UpgradeKymaOrchestration {
			_, count, _, err = operationsStorage.ListUpgradeKymaOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestrationExt.InProgress}})
		} else if orchestrationType == orchestrationExt.UpgradeClusterOrchestration {
			_, count, _, err = operationsStorage.ListUpgradeClusterOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestrationExt.InProgress}})
		}
		if err != nil {
			return fmt.Errorf("while listing %s operations for orchestration %s: %w", orchestrationType, o.OrchestrationID, err)
		}

		if count > 0 {
			log.Infof("Resuming the processing of %s %s orchestration ID: %s", orchestrationExt.Canceling, orchestrationType, o.OrchestrationID)
			queue.Add(o.OrchestrationID)
			return nil
		}
	}
	return nil
}

func initClient(cfg *rest.Config) (client.Client, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("while waiting for client mapper: %w", err)
		}
	}
	cli, err := client.New(cfg, client.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("while creating a client: %w", err)
	}
	return cli, nil
}

func fatalOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func skipForPreviewPlan(operation internal.Operation) bool {
	return !broker.IsPreviewPlan(operation.ProvisioningParameters.PlanID)
}
