package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	gruntime "runtime"
	"runtime/pprof"
	"sort"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"

	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	shoot "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	orchestrationExt "github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	eventshandler "github.com/kyma-project/kyma-environment-broker/internal/events/handler"
	"github.com/kyma-project/kyma-environment-broker/internal/health"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

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
	Database    storage.Config
	Gardener    gardener.Config
	Kubeconfig  kubeconfig.Config

	ManagedRuntimeComponentsYAMLFilePath       string
	NewAdditionalRuntimeComponentsYAMLFilePath string
	SkrOidcDefaultValuesYAMLFilePath           string
	SkrDnsProvidersValuesYAMLFilePath          string
	DefaultRequestRegion                       string `envconfig:"default=cf-eu10"`
	UpdateProcessingEnabled                    bool   `envconfig:"default=false"`
	LifecycleManagerIntegrationDisabled        bool   `envconfig:"default=true"`
	InfrastructureManagerIntegrationDisabled   bool   `envconfig:"default=true"`
	Broker                                     broker.Config
	CatalogFilePath                            string

	EDP edp.Config

	Notification notification.Config

	KymaDashboardConfig dashboard.Config

	OrchestrationConfig orchestration.Config

	TrialRegionMappingFilePath string

	SapConvergedCloudRegionMappingsFilePath string

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

	RuntimeConfigurationConfigMapName string `envconfig:"default=keb-runtime-config"`

	UpdateRuntimeResourceDelay time.Duration `envconfig:"default=4s"`
}

type ProfilerConfig struct {
	Path     string        `envconfig:"default=/tmp/profiler"`
	Sampling time.Duration `envconfig:"default=1s"`
	Memory   bool
}

type K8sClientProvider interface {
	K8sClientForRuntimeID(rid string) (client.Client, error)
	K8sClientSetForRuntimeID(runtimeID string) (*kubernetes.Clientset, error)
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
	err = imv1.AddToScheme(scheme.Scheme)
	panicOnError(err)
	err = shoot.AddToScheme(scheme.Scheme)
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

	cfg.OrchestrationConfig.KubernetesVersion = cfg.Provisioner.KubernetesVersion
	// create logger
	logger := lager.NewLogger("kyma-env-broker")

	logger.Info("Starting Kyma Environment Broker")

	logger.Info("Registering healthz endpoint for health probes")
	health.NewServer(cfg.Host, cfg.StatusPort, logs).ServeAsync()
	go periodicProfile(logger, cfg.Profiler)

	logConfiguration(logs, cfg)

	// create provisioner client
	provisionerClient := provisioner.NewProvisionerClient(cfg.Provisioner.URL, cfg.DumpProvisionerRequests, logs.WithField("service", "provisioner"))

	// create kubernetes client
	kcpK8sConfig, err := config.GetConfig()
	fatalOnError(err, logs)
	kcpK8sClient, err := initClient(kcpK8sConfig)
	fatalOnError(err, logs)
	skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(kcpK8sClient)

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

	// provides configuration for specified Kyma version and plan
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, kcpK8sClient, logs, cfg.RuntimeConfigurationConfigMapName),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	gardenerClusterConfig, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	fatalOnError(err, logs)
	cfg.Gardener.DNSProviders, err = gardener.ReadDNSProvidersValuesFromYAML(cfg.SkrDnsProvidersValuesYAMLFilePath)
	fatalOnError(err, logs)
	dynamicGardener, err := dynamic.NewForConfig(gardenerClusterConfig)
	fatalOnError(err, logs)
	gardenerClient, err := initClient(gardenerClusterConfig)
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
	inputFactory, err := input.NewInputBuilderFactory(configProvider, cfg.Provisioner, regions, cfg.FreemiumProviders, oidcDefaultValues, cfg.Broker.UseSmallerMachineTypes)
	fatalOnError(err, logs)

	edpClient := edp.NewClient(cfg.EDP)

	// application event broker
	eventBroker := event.NewPubSub(logs)

	// metrics collectors
	_ = metricsv2.Register(ctx, eventBroker, db.Operations(), db.Instances(), cfg.MetricsV2, logs)

	// run queues
	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Provisioning, logs.WithField("provisioning", "manager"))
	provisionQueue := NewProvisioningProcessingQueue(ctx, provisionManager, cfg.Provisioning.WorkersAmount, &cfg, db, provisionerClient, inputFactory,
		edpClient, accountProvider, skrK8sClientProvider, kcpK8sClient, oidcDefaultValues, logs)

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Deprovisioning, logs.WithField("deprovisioning", "manager"))
	deprovisionQueue := NewDeprovisioningProcessingQueue(ctx, cfg.Deprovisioning.WorkersAmount, deprovisionManager, &cfg, db, eventBroker, provisionerClient, edpClient, accountProvider,
		skrK8sClientProvider, kcpK8sClient, configProvider, logs)

	updateManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Update, logs.WithField("update", "manager"))
	updateQueue := NewUpdateProcessingQueue(ctx, updateManager, cfg.Update.WorkersAmount, db, inputFactory, provisionerClient, eventBroker,
		cfg, skrK8sClientProvider, kcpK8sClient, logs)
	/***/
	servicesConfig, err := broker.NewServicesConfigFromFile(cfg.CatalogFilePath)
	fatalOnError(err, logs)

	// create kubeconfig builder
	kcBuilder := kubeconfig.NewBuilder(provisionerClient, kcpK8sClient, skrK8sClientProvider)

	// create server
	router := mux.NewRouter()
	createAPI(router, servicesConfig, inputFactory, &cfg, db, provisionQueue, deprovisionQueue, updateQueue, logger, logs, inputFactory.GetPlanDefaults, kcBuilder, skrK8sClientProvider, skrK8sClientProvider, gardenerClient, kcpK8sClient)

	// create metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// create SKR kubeconfig endpoint
	kcHandler := kubeconfig.NewHandler(db, kcBuilder, cfg.Kubeconfig.AllowOrigins, broker.OwnClusterPlanID, logs.WithField("service", "kubeconfigHandle"))
	kcHandler.AttachRoutes(router)

	runtimeLister := orchestration.NewRuntimeLister(db.Instances(), db.Operations(), runtime.NewConverter(cfg.DefaultRequestRegion), logs)
	runtimeResolver := orchestrationExt.NewGardenerRuntimeResolver(dynamicGardener, gardenerNamespace, runtimeLister, logs)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory,
		nil, time.Minute, runtimeResolver, notificationBuilder, logs, kcpK8sClient, cfg, 1)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, clusterQueue, cfg.MaxPaginationPage, logs)

	if !cfg.DisableProcessOperationsInProgress {
		err = processOperationsInProgressByType(internal.OperationTypeProvision, db.Operations(), provisionQueue, logs)
		fatalOnError(err, logs)
		err = processOperationsInProgressByType(internal.OperationTypeDeprovision, db.Operations(), deprovisionQueue, logs)
		fatalOnError(err, logs)
		err = processOperationsInProgressByType(internal.OperationTypeUpdate, db.Operations(), updateQueue, logs)
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
		db.RuntimeStates(), db.InstancesArchived(), db.Bindings(), cfg.MaxPaginationPage,
		cfg.DefaultRequestRegion, provisionerClient,
		kcpK8sClient,
		cfg.Broker.KimConfig,
		logs)
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

func logConfiguration(logs *logrus.Logger, cfg Config) {
	logs.Infof("Setting provisioner timeouts: provisioning=%s, deprovisioning=%s", cfg.Provisioner.ProvisioningTimeout, cfg.Provisioner.DeprovisioningTimeout)
	logs.Infof("Setting staged manager configuration: provisioning=%s, deprovisioning=%s, update=%s", cfg.Provisioning, cfg.Deprovisioning, cfg.Update)
	logs.Infof("InfrastructureManagerIntegrationDisabled: %v", cfg.InfrastructureManagerIntegrationDisabled)
	logs.Infof("Archiving enabled: %v, dry run: %v", cfg.ArchiveEnabled, cfg.ArchiveDryRun)
	logs.Infof("Cleaning enabled: %v, dry run: %v", cfg.CleaningEnabled, cfg.CleaningDryRun)
	logs.Infof("KIM enabled: %t, dry run: %t, view only CR: %t, plans: %s, KIM only plans: %s",
		cfg.Broker.KimConfig.Enabled,
		cfg.Broker.KimConfig.DryRun,
		cfg.Broker.KimConfig.ViewOnly,
		cfg.Broker.KimConfig.Plans,
		cfg.Broker.KimConfig.KimOnlyPlans)
	logs.Infof("Is SubaccountMovementEnabled: %t", cfg.Broker.SubaccountMovementEnabled)
	logs.Infof("Is UpdateCustomResourcesLabelsOnAccountMove enabled: %t", cfg.Broker.UpdateCustomResourcesLabelsOnAccountMove)
}

func createAPI(router *mux.Router, servicesConfig broker.ServicesConfig, planValidator broker.PlanValidator, cfg *Config, db storage.BrokerStorage, provisionQueue, deprovisionQueue, updateQueue *process.Queue, logger lager.Logger, logs logrus.FieldLogger, planDefaults broker.PlanDefaults, kcBuilder kubeconfig.KcBuilder, clientProvider K8sClientProvider, kubeconfigProvider KubeconfigProvider, gardenerClient, kcpK8sClient client.Client) {
	suspensionCtxHandler := suspension.NewContextUpdateHandler(db.Operations(), provisionQueue, deprovisionQueue, logs)

	defaultPlansConfig, err := servicesConfig.DefaultPlansConfig()
	fatalOnError(err, logs)

	debugSink, err := lager.NewRedactingSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), []string{"instance-details"}, []string{})
	fatalOnError(err, logs)
	logger.RegisterSink(debugSink)
	errorSink, err := lager.NewRedactingSink(lager.NewWriterSink(os.Stderr, lager.ERROR), []string{"instance-details"}, []string{})
	fatalOnError(err, logs)
	logger.RegisterSink(errorSink)

	freemiumGlobalAccountIds, err := whitelist.ReadWhitelistedGlobalAccountIdsFromFile(cfg.FreemiumWhitelistedGlobalAccountsFilePath)
	fatalOnError(err, logs)
	logs.Infof("Number of globalAccountIds for unlimited freeemium: %d\n", len(freemiumGlobalAccountIds))

	// backward compatibility for tests
	convergedCloudRegionProvider, err := broker.NewDefaultConvergedCloudRegionsProvider(cfg.SapConvergedCloudRegionMappingsFilePath, &broker.YamlRegionReader{})
	fatalOnError(err, logs)
	logs.Infof("%s plan region mappings loaded", broker.SapConvergedCloudPlanName)

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		ServicesEndpoint: broker.NewServices(cfg.Broker, servicesConfig, logs, convergedCloudRegionProvider),
		ProvisionEndpoint: broker.NewProvision(cfg.Broker, cfg.Gardener, db.Operations(), db.Instances(), db.InstancesArchived(),
			provisionQueue, planValidator, defaultPlansConfig,
			planDefaults, logs, cfg.KymaDashboardConfig, kcBuilder, freemiumGlobalAccountIds, convergedCloudRegionProvider,
		),
		DeprovisionEndpoint: broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		UpdateEndpoint: broker.NewUpdate(cfg.Broker, db.Instances(), db.RuntimeStates(), db.Operations(),
			suspensionCtxHandler, cfg.UpdateProcessingEnabled, cfg.Broker.SubaccountMovementEnabled, cfg.Broker.UpdateCustomResourcesLabelsOnAccountMove, updateQueue, defaultPlansConfig,
			planDefaults, logs, cfg.KymaDashboardConfig, kcBuilder, convergedCloudRegionProvider, kcpK8sClient),
		GetInstanceEndpoint:          broker.NewGetInstance(cfg.Broker, db.Instances(), db.Operations(), kcBuilder, logs),
		LastOperationEndpoint:        broker.NewLastOperation(db.Operations(), db.InstancesArchived(), logs),
		BindEndpoint:                 broker.NewBind(cfg.Broker.Binding, db.Instances(), db.Bindings(), logs, clientProvider, kubeconfigProvider, gardenerClient),
		UnbindEndpoint:               broker.NewUnbind(logs),
		GetBindingEndpoint:           broker.NewGetBinding(logs, db.Bindings()),
		LastBindingOperationEndpoint: broker.NewLastBindingOperation(logs),
	}

	router.Use(middleware.AddRequestLogging(logs))
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
		if orchestrationType == orchestrationExt.UpgradeClusterOrchestration {
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
