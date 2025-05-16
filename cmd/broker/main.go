package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	gruntime "runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/additionalproperties"
	"github.com/kyma-project/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	brokerBindings "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	eventshandler "github.com/kyma-project/kyma-environment-broker/internal/events/handler"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/health"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/regionssupportingmachine"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/suspension"
	"github.com/kyma-project/kyma-environment-broker/internal/swagger"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	shoot "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vrischmann/envconfig"
	"golang.org/x/exp/maps"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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

	InfrastructureManager broker.InfrastructureManager
	Database              storage.Config
	Gardener              gardener.Config
	Kubeconfig            kubeconfig.Config
	StepTimeouts          StepTimeoutsConfig

	SkrOidcDefaultValuesYAMLFilePath    string
	SkrDnsProvidersValuesYAMLFilePath   string
	DefaultRequestRegion                string `envconfig:"default=cf-eu10"`
	UpdateProcessingEnabled             bool   `envconfig:"default=false"`
	LifecycleManagerIntegrationDisabled bool   `envconfig:"default=true"`
	Broker                              broker.Config
	CatalogFilePath                     string

	EDP edp.Config

	KymaDashboardConfig dashboard.Config

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

	RuntimeConfigurationConfigMapName string `envconfig:"default=keb-runtime-config"`

	UpdateRuntimeResourceDelay time.Duration `envconfig:"default=4s"`

	RegionsSupportingMachineFilePath string

	HapRuleFilePath string

	MultipleContexts bool `envconfig:"default=false"`

	ProvidersConfigurationFilePath string

	PlansConfigurationFilePath string
}

type ProfilerConfig struct {
	Path     string        `envconfig:"default=/tmp/profiler"`
	Sampling time.Duration `envconfig:"default=1s"`
	Memory   bool
}

type StepTimeoutsConfig struct {
	CheckRuntimeResourceCreate   time.Duration `envconfig:"default=60m"`
	CheckRuntimeResourceUpdate   time.Duration `envconfig:"default=180m"`
	CheckRuntimeResourceDeletion time.Duration `envconfig:"default=1h"`
}

type K8sClientProvider interface {
	K8sClientForRuntimeID(rid string) (client.Client, error)
	K8sClientSetForRuntimeID(runtimeID string) (kubernetes.Interface, error)
}

type KubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeId string) ([]byte, error)
}

const (
	createRuntimeStageName      = "create_runtime"
	checkKymaStageName          = "check_kyma"
	createKymaResourceStageName = "create_kyma_resource"
	startStageName              = "start"
	brokerAPISubrouterName      = "brokerAPI"
)

func periodicProfile(logger *slog.Logger, profiler ProfilerConfig) {
	if profiler.Memory == false {
		return
	}
	logger.Info(fmt.Sprintf("Starting periodic profiler %v", profiler))
	if err := os.MkdirAll(profiler.Path, os.ModePerm); err != nil {
		logger.Error(fmt.Sprintf("Failed to create dir %v for profile storage: %v", profiler.Path, err))
	}
	for {
		profName := fmt.Sprintf("%v/mem-%v.pprof", profiler.Path, time.Now().Unix())
		logger.Info(fmt.Sprintf("Creating periodic memory profile %v", profName))
		profFile, err := os.Create(profName)
		if err != nil {
			logger.Error(fmt.Sprintf("Creating periodic memory profile %v failed: %v", profName, err))
		}
		err = pprof.Lookup("allocs").WriteTo(profFile, 0)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to write periodic memory profile to %v file: %v", profName, err))
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
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelInfo)
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(log)

	// create and fill config
	var cfg Config
	err = envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err, log)

	if cfg.LogLevel != "" {
		logLevel.Set(cfg.getLogLevel())
	}

	// create logger
	logger := lager.NewLogger("kyma-env-broker")

	log.Info("Starting Kyma Environment Broker")

	log.Info("Registering healthz endpoint for health probes")
	health.NewServer(cfg.Host, cfg.StatusPort, log).ServeAsync()
	go periodicProfile(log, cfg.Profiler)

	logConfiguration(log, cfg)

	// create kubernetes client
	kcpK8sConfig, err := config.GetConfig()
	fatalOnError(err, log)
	kcpK8sClient, err := initClient(kcpK8sConfig)
	fatalOnError(err, log)
	skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(kcpK8sClient)

	if cfg.Broker.MonitorAdditionalProperties {
		err := os.MkdirAll(cfg.Broker.AdditionalPropertiesPath, os.ModePerm)
		fatalOnError(err, log)
	}

	// create storage
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	var db storage.BrokerStorage
	if cfg.DbInMemory {
		db = storage.NewMemoryStorage()
	} else {
		store, conn, err := storage.NewFromConfig(cfg.Database, cfg.Events, cipher)
		fatalOnError(err, log)
		db = store
		dbStatsCollector := sqlstats.NewStatsCollector("broker", conn)
		prometheus.MustRegister(dbStatsCollector)
	}

	// provides configuration for specified Kyma version and plan
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, kcpK8sClient, log, cfg.RuntimeConfigurationConfigMapName),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	gardenerClusterConfig, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	fatalOnError(err, log)
	cfg.Gardener.DNSProviders, err = gardener.ReadDNSProvidersValuesFromYAML(cfg.SkrDnsProvidersValuesYAMLFilePath)
	fatalOnError(err, log)
	dynamicGardener, err := dynamic.NewForConfig(gardenerClusterConfig)
	fatalOnError(err, log)

	gardenerNamespace := fmt.Sprintf("garden-%v", cfg.Gardener.Project)
	gardenerAccountPool := hyperscaler.NewAccountPool(dynamicGardener, gardenerNamespace)
	gardenerSharedPool := hyperscaler.NewSharedGardenerAccountPool(dynamicGardener, gardenerNamespace)
	accountProvider := hyperscaler.NewAccountProvider(gardenerAccountPool, gardenerSharedPool)
	gardenerClient := gardener.NewClient(dynamicGardener, gardenerNamespace)

	oidcDefaultValues, err := runtime.ReadOIDCDefaultValuesFromYAML(cfg.SkrOidcDefaultValuesYAMLFilePath)
	fatalOnError(err, log)

	edpClient := edp.NewClient(cfg.EDP)

	// application event broker
	eventBroker := event.NewPubSub(log)

	// metrics collectors
	_ = metricsv2.Register(ctx, eventBroker, db, cfg.MetricsV2, log)

	rulesService, err := rules.NewRulesServiceFromFile(cfg.HapRuleFilePath, sets.New(maps.Keys(broker.PlanIDsMapping)...), sets.New([]string(cfg.Broker.EnablePlans)...).Delete("own_cluster"))
	fatalOnError(err, log)

	rulesetValid := rulesService.IsRulesetValid()

	if !rulesetValid {
		log.Error("There are errors in subscription secret rules configuration:")
		for _, ve := range rulesService.ValidationInfo.All() {
			log.Error(fmt.Sprintf("%s", ve))
		}
		fatalOnError(err, log)
	}

	regionsSupportingMachine, err := regionssupportingmachine.ReadRegionsSupportingMachineFromFile(cfg.RegionsSupportingMachineFilePath)
	fatalOnError(err, log)
	log.Info(fmt.Sprintf("Number of machine types families that are not universally supported across all regions: %d", len(regionsSupportingMachine)))

	workersProvider := workers.NewProvider(cfg.InfrastructureManager, regionsSupportingMachine)

	plansConfiguration, err := os.Open(cfg.PlansConfigurationFilePath)
	fatalOnError(err, log)
	providersConfiguration, err := os.Open(cfg.ProvidersConfigurationFilePath)
	fatalOnError(err, log)
	schemaService, err := broker.NewSchemaService(providersConfiguration, plansConfiguration, &oidcDefaultValues, cfg.Broker, cfg.InfrastructureManager.EnableIngressFiltering, cfg.InfrastructureManager.IngressFilteringPlans)
	fatalOnError(err, log)
	fatalOnError(schemaService.Validate(), log)
	log.Info("Plans and providers configuration is valid")

	// run queues
	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Provisioning, log.With("provisioning", "manager"))
	provisionQueue := NewProvisioningProcessingQueue(ctx, provisionManager, cfg.Provisioning.WorkersAmount, &cfg, db, configProvider,
		edpClient, accountProvider, skrK8sClientProvider, kcpK8sClient, gardenerClient, oidcDefaultValues, log, rulesService, workersProvider)

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Deprovisioning, log.With("deprovisioning", "manager"))
	deprovisionQueue := NewDeprovisioningProcessingQueue(ctx, cfg.Deprovisioning.WorkersAmount, deprovisionManager, &cfg, db, edpClient, accountProvider,
		skrK8sClientProvider, kcpK8sClient, configProvider, log)

	updateManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, cfg.Update, log.With("update", "manager"))
	updateQueue := NewUpdateProcessingQueue(ctx, updateManager, cfg.Update.WorkersAmount, db, cfg, kcpK8sClient, log, workersProvider, schemaService)
	/***/
	servicesConfig, err := broker.NewServicesConfigFromFile(cfg.CatalogFilePath)
	fatalOnError(err, log)

	// create kubeconfig builder
	kcBuilder := kubeconfig.NewBuilder(kcpK8sClient, skrK8sClientProvider, cfg.MultipleContexts)

	// create server
	router := httputil.NewRouter()

	createAPI(router, schemaService, servicesConfig, &cfg, db, provisionQueue, deprovisionQueue, updateQueue, logger, log,
		kcBuilder, skrK8sClientProvider, skrK8sClientProvider, kcpK8sClient, eventBroker, oidcDefaultValues, regionsSupportingMachine)

	// create metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// create SKR kubeconfig endpoint
	kcHandler := kubeconfig.NewHandler(db, kcBuilder, cfg.Kubeconfig.AllowOrigins, broker.OwnClusterPlanID, log.With("service", "kubeconfigHandle"))
	kcHandler.AttachRoutes(router)

	if !cfg.DisableProcessOperationsInProgress {
		err = processOperationsInProgressByType(internal.OperationTypeProvision, db.Operations(), provisionQueue, log)
		fatalOnError(err, log)
		err = processOperationsInProgressByType(internal.OperationTypeDeprovision, db.Operations(), deprovisionQueue, log)
		fatalOnError(err, log)
		err = processOperationsInProgressByType(internal.OperationTypeUpdate, db.Operations(), updateQueue, log)
		fatalOnError(err, log)
	} else {
		log.Info("Skipping processing operation in progress on start")
	}

	// configure templates e.g. {{.domain}} to replace it with the domain name
	swaggerTemplates := map[string]string{
		"domain": cfg.DomainName,
	}
	err = swagger.NewTemplate("/swagger", swaggerTemplates).Execute()
	fatalOnError(err, log)

	// create list runtimes endpoint
	runtimeHandler := runtime.NewHandler(db, cfg.MaxPaginationPage,
		cfg.DefaultRequestRegion,
		kcpK8sClient,
		log)
	runtimeHandler.AttachRoutes(router)

	// create list requests with additional properties endpoint
	additionalPropertiesHandler := additionalproperties.NewHandler(log, cfg.Broker.AdditionalPropertiesPath)
	additionalPropertiesHandler.AttachRoutes(router)

	// create expiration endpoint
	expirationHandler := expiration.NewHandler(db.Instances(), db.Operations(), deprovisionQueue, log)
	expirationHandler.AttachRoutes(router)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/", http.FileServer(http.Dir("/swagger"))).ServeHTTP(w, r)
	})

	svr := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httputil.NewResponseRecorder(w)
		router.ServeHTTP(rec, r)
		log.Info(fmt.Sprintf("Call handled: method=%s url=%s statusCode=%d size=%d", r.Method, r.URL.Path, rec.StatusCode, rec.Size))
	})
	fatalOnError(http.ListenAndServe(cfg.Host+":"+cfg.Port, svr), log)
}

func logConfiguration(logs *slog.Logger, cfg Config) {
	logs.Info(fmt.Sprintf("Setting staged manager configuration: provisioning=%s, deprovisioning=%s, update=%s", cfg.Provisioning, cfg.Deprovisioning, cfg.Update))
	logs.Info(fmt.Sprintf("EnablePlans: %s", cfg.Broker.EnablePlans))
	logs.Info(fmt.Sprintf("Archiving enabled: %v, dry run: %v", cfg.ArchiveEnabled, cfg.ArchiveDryRun))
	logs.Info(fmt.Sprintf("Cleaning enabled: %v, dry run: %v", cfg.CleaningEnabled, cfg.CleaningDryRun))
	logs.Info(fmt.Sprintf("Is SubaccountMovementEnabled: %t", cfg.Broker.SubaccountMovementEnabled))
	logs.Info(fmt.Sprintf("Is UpdateCustomResourcesLabelsOnAccountMove enabled: %t", cfg.Broker.UpdateCustomResourcesLabelsOnAccountMove))
	logs.Info(fmt.Sprintf("StepTimeouts: CheckRuntimeResourceCreate=%s, CheckRuntimeResourceUpdate=%s, CheckRuntimeResourceDeletion=%s", cfg.StepTimeouts.CheckRuntimeResourceCreate, cfg.StepTimeouts.CheckRuntimeResourceUpdate, cfg.StepTimeouts.CheckRuntimeResourceDeletion))

	logs.Info(fmt.Sprintf("InfrastructureManager.Kubernetes Version: %s", cfg.InfrastructureManager.KubernetesVersion))
	logs.Info(fmt.Sprintf("InfrastructureManager.DefaultGardenerShootPurpose: %s", cfg.InfrastructureManager.DefaultGardenerShootPurpose))
	logs.Info(fmt.Sprintf("InfrastructureManager.MachineImage: %s", cfg.InfrastructureManager.MachineImage))
	logs.Info(fmt.Sprintf("InfrastructureManager.MachineImageVersion: %s", cfg.InfrastructureManager.MachineImageVersion))
	logs.Info(fmt.Sprintf("InfrastructureManager.DefaultTrialProvider: %s", cfg.InfrastructureManager.DefaultTrialProvider))
	logs.Info(fmt.Sprintf("InfrastructureManager.MultiZoneCluster: %v", cfg.InfrastructureManager.MultiZoneCluster))
	logs.Info(fmt.Sprintf("InfrastructureManager.ControlPlaneFailureTolerance: %s", cfg.InfrastructureManager.ControlPlaneFailureTolerance))
	logs.Info(fmt.Sprintf("InfrastructureManager.UseSmallerMachineTypes: %v", cfg.InfrastructureManager.UseSmallerMachineTypes))
	logs.Info(fmt.Sprintf("InfrastructureManager.EnableIngressFiltering: %v", cfg.InfrastructureManager.EnableIngressFiltering))
	logs.Info(fmt.Sprintf("InfrastructureManager.IngressFilteringPlans: %s", cfg.InfrastructureManager.IngressFilteringPlans))
}

func createAPI(router *httputil.Router, schemaService *broker.SchemaService, servicesConfig broker.ServicesConfig, cfg *Config, db storage.BrokerStorage,
	provisionQueue, deprovisionQueue, updateQueue *process.Queue, logger lager.Logger, logs *slog.Logger, kcBuilder kubeconfig.KcBuilder, clientProvider K8sClientProvider,
	kubeconfigProvider KubeconfigProvider, kcpK8sClient client.Client, publisher event.Publisher, oidcDefaultValues pkg.OIDCConfigDTO, regionsSupportingMachine regionssupportingmachine.RegionsSupportingMachine) {

	regions, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	fatalOnError(err, logs)
	logs.Info(fmt.Sprintf("Platform region mapping for trial: %v", regions))
	valuesProvider := provider.NewPlanSpecificValuesProvider(cfg.InfrastructureManager, regions, schemaService)

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
	logs.Info(fmt.Sprintf("Number of globalAccountIds for unlimited freemium: %d", len(freemiumGlobalAccountIds)))

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		ServicesEndpoint: broker.NewServices(cfg.Broker, schemaService, servicesConfig, logs, oidcDefaultValues, cfg.InfrastructureManager),
		ProvisionEndpoint: broker.NewProvision(cfg.Broker, cfg.Gardener, cfg.InfrastructureManager, db,
			provisionQueue, defaultPlansConfig, logs, cfg.KymaDashboardConfig, kcBuilder, freemiumGlobalAccountIds,
			schemaService, regionsSupportingMachine, valuesProvider, cfg.InfrastructureManager.UseSmallerMachineTypes,
		),
		DeprovisionEndpoint: broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		UpdateEndpoint: broker.NewUpdate(cfg.Broker, db,
			suspensionCtxHandler, cfg.UpdateProcessingEnabled, cfg.Broker.SubaccountMovementEnabled, cfg.Broker.UpdateCustomResourcesLabelsOnAccountMove, updateQueue, defaultPlansConfig,
			valuesProvider, logs, cfg.KymaDashboardConfig, kcBuilder, kcpK8sClient, regionsSupportingMachine, cfg.InfrastructureManager, schemaService),
		GetInstanceEndpoint:          broker.NewGetInstance(cfg.Broker, db.Instances(), db.Operations(), kcBuilder, logs),
		LastOperationEndpoint:        broker.NewLastOperation(db.Operations(), db.InstancesArchived(), logs),
		BindEndpoint:                 broker.NewBind(cfg.Broker.Binding, db, logs, clientProvider, kubeconfigProvider, publisher, cfg.MultipleContexts),
		UnbindEndpoint:               broker.NewUnbind(logs, db, brokerBindings.NewServiceAccountBindingsManager(clientProvider, kubeconfigProvider, cfg.MultipleContexts), publisher),
		GetBindingEndpoint:           broker.NewGetBinding(logs, db),
		LastBindingOperationEndpoint: broker.NewLastBindingOperation(logs),
	}

	prefixes := []string{"/{region}", ""}
	subRouter, err := router.NewSubRouter(brokerAPISubrouterName)
	fatalOnError(err, logs)
	broker.AttachRoutes(subRouter, kymaEnvBroker, logs, cfg.Broker.Binding.CreateBindingTimeout, cfg.DefaultRequestRegion, prefixes)
	router.Handle("/oauth/", http.StripPrefix("/oauth", subRouter))

	respWriter := httputil.NewResponseWriter(logs, cfg.DevelopmentMode)
	runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(db.Instances(), db.Operations(), defaultPlansConfig, cfg.DefaultRequestRegion, respWriter)
	router.Handle("/info/runtimes", runtimesInfoHandler)
	router.Handle("/events", eventshandler.NewHandler(db.Events(), db.Instances()))
}

// queues all in progress operations by type
func processOperationsInProgressByType(opType internal.OperationType, op storage.Operations, queue *process.Queue, log *slog.Logger) error {
	operations, err := op.GetNotFinishedOperationsByType(opType)
	if err != nil {
		return fmt.Errorf("while getting in progress operations from storage: %w", err)
	}
	for _, operation := range operations {
		queue.Add(operation.ID)
		log.Info(fmt.Sprintf("Resuming the processing of %s operation ID: %s", opType, operation.ID))
	}
	return nil
}

func initClient(cfg *rest.Config) (client.Client, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, false, func(ctx context.Context) (bool, error) {
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

func fatalOnError(err error, log *slog.Logger) {
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func (c Config) getLogLevel() slog.Level {
	switch strings.ToUpper(c.LogLevel) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
