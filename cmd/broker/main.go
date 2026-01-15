package main

import (
	"context"
	"crypto/fips140"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	gruntime "runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/additionalproperties"
	"github.com/kyma-project/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	brokerBindings "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	"github.com/kyma-project/kyma-environment-broker/internal/btpregionsmigration"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	eventshandler "github.com/kyma-project/kyma-environment-broker/internal/events/handler"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/health"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/machinesavailability"
	"github.com/kyma-project/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/quota"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/suspension"
	"github.com/kyma-project/kyma-environment-broker/internal/swagger"
	"github.com/kyma-project/kyma-environment-broker/internal/version"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	shoot "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vrischmann/envconfig"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	InfrastructureManager broker.InfrastructureManager
	Database              storage.Config
	Gardener              gardener.Config
	Kubeconfig            kubeconfig.Config
	StepTimeouts          StepTimeoutsConfig

	SkrOidcDefaultValuesYAMLFilePath  string
	SkrDnsProvidersValuesYAMLFilePath string
	UpdateProcessingEnabled           bool `envconfig:"default=false"`
	Broker                            broker.Config
	CatalogFilePath                   string

	KymaDashboardConfig dashboard.Config

	TrialRegionMappingFilePath string

	MaxPaginationPage int `envconfig:"default=100"`

	LogLevel string `envconfig:"default=info"`

	FreemiumWhitelistedGlobalAccountsFilePath string

	DomainName string

	// Enable/disable profiler configuration. The profiler samples will be stored
	// under /tmp/profiler directory. Based on the deployment strategy, this will be
	// either ephemeral container filesystem or persistent storage
	Profiler ProfilerConfig

	Events events.Config

	Metrics metrics.Config

	Provisioning   process.StagedManagerConfiguration
	Deprovisioning process.StagedManagerConfiguration
	Update         process.StagedManagerConfiguration

	RuntimeConfigurationConfigMapName string `envconfig:"default=keb-runtime-config"`

	UpdateRuntimeResourceDelay time.Duration `envconfig:"default=4s"`

	RegionsSupportingMachineFilePath string

	HapRuleFilePath string

	ProvidersConfigurationFilePath string

	PlansConfigurationFilePath string

	// allows to configure which k8s resource in the Gardener must be used for HAP and discovery zones feature
	SubscriptionGardenerResource string `envconfig:"default=SecretBinding"`

	Quota                               quota.Config
	QuotaWhitelistedSubaccountsFilePath string

	// todo: remove after all SecretBinding are migrated to CredentialBinding resources
	HoldHapSteps bool

	MachinesAvailabilityEndpoint bool

	BtpRegionsMigrationSapConvergedCloudFilePath string
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
	brokerAPISubrouterName         = "brokerAPI"
	provisioningTakesLongThreshold = 20 * time.Minute
)

var Version string

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

func (c *Config) Validate() error {
	_, err := c.GardenerSubscriptionResource()
	return err
}

func (c *Config) GardenerSubscriptionResource() (schema.GroupVersionResource, error) {
	resourceName := strings.ToLower(c.SubscriptionGardenerResource)
	switch resourceName {
	case "secretbinding":
		return gardener.SecretBindingResource, nil
	case "credentialsbinding":
		return gardener.CredentialsBindingResource, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("invalid SubscriptionGardenerResource: %s. Supported values are SecretBinding and CredentialsBinding", c.SubscriptionGardenerResource)
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

	if broker.AvailablePlans == nil {
		fatalOnError(fmt.Errorf("AvailablePlans is not initialized properly"), log)
	}

	log.Info("Starting Kyma Environment Broker")

	log.Info("Registering healthz endpoint for health probes")
	health.NewServer(cfg.Broker.Host, cfg.Broker.StatusPort, log).ServeAsync()
	go periodicProfile(log, cfg.Profiler)

	logConfiguration(log, cfg)

	if fips140.Enabled() {
		log.Info("FIPS mode is enabled")
	} else {
		log.Info("FIPS mode is disabled")
	}

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

	cipher := storage.NewEncrypter(cfg.Database.SecretKey)

	// create storage
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

	// get storage time zone
	timeZone, err := db.TimeZones().GetTimeZone()
	fatalOnError(err, log)
	log.Info(fmt.Sprintf("KEB local time: %s time zone: %s", time.Now().String(), time.Now().Location().String()))
	log.Info(fmt.Sprintf("Storage time zone: %s", timeZone))

	// provides configuration for specified Kyma version and plan
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, kcpK8sClient, log),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	gardenerClusterConfig, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	fatalOnError(err, log)
	cfg.Gardener.DNSProviders, err = gardener.ReadDNSProvidersValuesFromYAML(cfg.SkrDnsProvidersValuesYAMLFilePath)
	fatalOnError(err, log)
	dynamicGardener, err := dynamic.NewForConfig(gardenerClusterConfig)
	fatalOnError(err, log)

	gardenerNamespace := fmt.Sprintf("garden-%v", cfg.Gardener.Project)
	gardenerClient := gardener.NewClient(dynamicGardener, gardenerNamespace)

	oidcDefaultValues, err := runtime.ReadOIDCDefaultValuesFromYAML(cfg.SkrOidcDefaultValuesYAMLFilePath)
	fatalOnError(err, log)

	// application event broker
	eventBroker := event.NewPubSub(log)

	// metrics collectors
	_ = metrics.Register(ctx, eventBroker, db, cfg.Metrics, log)

	rulesService, err := rules.NewRulesServiceFromFile(cfg.HapRuleFilePath, sets.New(broker.AvailablePlans.GetAllPlanNamesAsStrings()...), sets.New([]string(cfg.Broker.EnablePlans)...))
	fatalOnError(err, log)

	rulesetValid := rulesService.IsRulesetValid()

	if !rulesetValid {
		log.Error("There are errors in subscription secret rules configuration:")
		for _, ve := range rulesService.ValidationInfo.All() {
			log.Error(fmt.Sprintf("%s", ve))
		}
		fatalOnError(err, log)
	}

	plansSpec, err := configuration.NewPlanSpecificationsFromFile(cfg.PlansConfigurationFilePath)
	fatalOnError(err, log)
	providerSpec, err := configuration.NewProviderSpecFromFile(cfg.ProvidersConfigurationFilePath)
	fatalOnError(err, log)
	fatalOnError(providerSpec.ValidateZonesDiscovery(), log)

	runtimeConfigProvider := kebConfig.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, kebConfig.RuntimeConfigurationRequiredFields)
	channelResolver, err := kebConfig.NewChannelResolver(runtimeConfigProvider, broker.AvailablePlans.GetAllPlanNamesAsStrings(), log)
	fatalOnError(err, log)

	schemaService := broker.NewSchemaService(providerSpec, plansSpec, &oidcDefaultValues, cfg.Broker, cfg.InfrastructureManager.IngressFilteringPlans, channelResolver)
	fatalOnError(err, log)
	fatalOnError(schemaService.Validate(), log)
	log.Info("Plans and providers configuration is valid")
	workersProvider := workers.NewProvider(cfg.InfrastructureManager, providerSpec)

	awsClientFactory := aws.NewFactory()

	// run queues
	provisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.Broker.OperationTimeout, cfg.Provisioning, log.With("provisioning", "manager"))
	provisionQueue := NewProvisioningProcessingQueue(ctx, provisionManager, cfg.Provisioning.WorkersAmount, &cfg, db, configProvider,
		skrK8sClientProvider, kcpK8sClient, gardenerClient, oidcDefaultValues, log, rulesService, workersProvider, providerSpec, awsClientFactory)

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.Broker.OperationTimeout, cfg.Deprovisioning, log.With("deprovisioning", "manager"))
	deprovisionQueue := NewDeprovisioningProcessingQueue(ctx, cfg.Deprovisioning.WorkersAmount, deprovisionManager, &cfg, db,
		skrK8sClientProvider, kcpK8sClient, configProvider, dynamicGardener, gardenerNamespace, log)

	updateManager := process.NewStagedManager(db.Operations(), eventBroker, cfg.Broker.OperationTimeout, cfg.Update, log.With("update", "manager"))
	updateQueue := NewUpdateProcessingQueue(ctx, updateManager, cfg.Update.WorkersAmount, db, cfg, kcpK8sClient, log, workersProvider, schemaService, plansSpec, configProvider, providerSpec, gardenerClient, awsClientFactory)
	/***/
	servicesConfig, err := broker.NewServicesConfigFromFile(cfg.CatalogFilePath)
	fatalOnError(err, log)

	// create kubeconfig builder
	kcBuilder := kubeconfig.NewBuilder(kcpK8sClient, skrK8sClientProvider)

	// create server
	router := httputil.NewRouter()

	// Apply panic recovery middleware to all HTTP endpoints
	router.Use(httputil.PanicRecoveryMiddleware(log))

	createAPI(router, schemaService, servicesConfig, &cfg, db, provisionQueue, deprovisionQueue, updateQueue, logger, log,
		kcBuilder, skrK8sClientProvider, skrK8sClientProvider, kcpK8sClient, eventBroker, oidcDefaultValues,
		providerSpec, configProvider, plansSpec, rulesService, gardenerClient, awsClientFactory)

	// create metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// create SKR kubeconfig endpoint
	kcHandler := kubeconfig.NewHandler(db, kcBuilder, cfg.Kubeconfig.AllowOrigins, log.With("service", "kubeconfigHandle"))
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
		cfg.Broker.DefaultRequestRegion,
		kcpK8sClient,
		log)
	router.HandleFunc("/runtimes", runtimeHandler.GetRuntimes)

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
	fatalOnError(http.ListenAndServe(cfg.Broker.Host+":"+cfg.Broker.Port, svr), log)
}

func logConfiguration(logs *slog.Logger, cfg Config) {
	logs.Info(fmt.Sprintf("Setting staged manager configuration: provisioning=%s, deprovisioning=%s, update=%s", cfg.Provisioning, cfg.Deprovisioning, cfg.Update))
	logs.Info(fmt.Sprintf("EnablePlans: %s", cfg.Broker.EnablePlans))
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
	logs.Info(fmt.Sprintf("InfrastructureManager.IngressFilteringPlans: %s", cfg.InfrastructureManager.IngressFilteringPlans))

	r, _ := cfg.GardenerSubscriptionResource()
	logs.Info(fmt.Sprintf("Gardener resource used for subscriptions: %s", r.String()))
}

func createAPI(router *httputil.Router, schemaService *broker.SchemaService, servicesConfig broker.ServicesConfig, cfg *Config, db storage.BrokerStorage,
	provisionQueue, deprovisionQueue, updateQueue *process.Queue, logger lager.Logger, logs *slog.Logger, kcBuilder kubeconfig.KcBuilder, clientProvider K8sClientProvider,
	kubeconfigProvider KubeconfigProvider, kcpK8sClient client.Client, publisher event.Publisher, oidcDefaultValues pkg.OIDCConfigDTO,
	providerSpec *configuration.ProviderSpec, configProvider kebConfig.Provider, planSpec *configuration.PlanSpecifications, rulesService *rules.RulesService,
	gardenerClient *gardener.Client, awsClientFactory aws.ClientFactory) {

	if cfg.MachinesAvailabilityEndpoint {
		if r, _ := cfg.GardenerSubscriptionResource(); r == gardener.SecretBindingResource {
			machinesAvailability := machinesavailability.NewHandler(providerSpec, rulesService, gardenerClient, awsClientFactory, logs)
			machinesAvailability.AttachRoutes(router)
		} else {
			machinesAvailability := machinesavailability.NewHandlerCB(providerSpec, rulesService, gardenerClient, awsClientFactory, logs)
			machinesAvailability.AttachRoutes(router)
		}
	}

	regions, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	fatalOnError(err, logs)
	logs.Info(fmt.Sprintf("Platform region mapping for trial: %v", regions))
	valuesProvider := provider.NewPlanSpecificValuesProvider(cfg.InfrastructureManager, regions, schemaService, planSpec)

	suspensionCtxHandler := suspension.NewContextUpdateHandler(db.Operations(), provisionQueue, deprovisionQueue, logs)

	defaultPlansConfig, err := servicesConfig.DefaultPlansConfig()
	fatalOnError(err, logs)

	debugSink, err := lager.NewRedactingSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), []string{"instance-details"}, []string{})
	fatalOnError(err, logs)
	logger.RegisterSink(debugSink)
	errorSink, err := lager.NewRedactingSink(lager.NewWriterSink(os.Stderr, lager.ERROR), []string{"instance-details"}, []string{})
	fatalOnError(err, logs)
	logger.RegisterSink(errorSink)

	freemiumGlobalAccountIds, err := whitelist.ReadWhitelistedIdsFromFile(cfg.FreemiumWhitelistedGlobalAccountsFilePath)
	fatalOnError(err, logs)
	logs.Info(fmt.Sprintf("Number of globalAccountIds for unlimited freemium: %d", len(freemiumGlobalAccountIds)))

	quotaClient := quota.NewClient(context.Background(), cfg.Quota, logs)
	quotaWhitelistedSubaccountIds, err := whitelist.ReadWhitelistedIdsFromFile(cfg.QuotaWhitelistedSubaccountsFilePath)
	fatalOnError(err, logs)
	logs.Info(fmt.Sprintf("Number of subaccountIds with unlimited quota: %d", len(quotaWhitelistedSubaccountIds)))

	btpRegionsMigrationSapConvergedCloud, err := btpregionsmigration.ReadFromFile(cfg.BtpRegionsMigrationSapConvergedCloudFilePath)
	fatalOnError(err, logs)
	logs.Info(fmt.Sprintf("Number of deprecated BTP regions for SAP Converged Cloud: %d", len(btpRegionsMigrationSapConvergedCloud)))

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		ServicesEndpoint: broker.NewServices(cfg.Broker, schemaService, servicesConfig, logs, oidcDefaultValues, cfg.InfrastructureManager),
		ProvisionEndpoint: broker.NewProvision(cfg.Broker, cfg.Gardener, cfg.InfrastructureManager, db,
			provisionQueue, defaultPlansConfig, logs, cfg.KymaDashboardConfig, kcBuilder, freemiumGlobalAccountIds,
			schemaService, providerSpec, valuesProvider, cfg.InfrastructureManager.UseSmallerMachineTypes,
			kebConfig.NewConfigMapConfigProvider(configProvider, cfg.Broker.GardenerSeedsCacheConfigMapName, kebConfig.ProviderConfigurationRequiredFields), quotaClient, quotaWhitelistedSubaccountIds,
			rulesService, gardenerClient, awsClientFactory, btpRegionsMigrationSapConvergedCloud),
		DeprovisionEndpoint: broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		UpdateEndpoint: broker.NewUpdate(cfg.Broker, db,
			suspensionCtxHandler, cfg.UpdateProcessingEnabled, cfg.Broker.SubaccountMovementEnabled, cfg.Broker.UpdateCustomResourcesLabelsOnAccountMove, updateQueue, defaultPlansConfig,
			valuesProvider, logs, cfg.KymaDashboardConfig, kcBuilder, kcpK8sClient, providerSpec, planSpec, cfg.InfrastructureManager, schemaService, quotaClient, quotaWhitelistedSubaccountIds,
			rulesService, gardenerClient, awsClientFactory),
		GetInstanceEndpoint:          broker.NewGetInstance(cfg.Broker, db.Instances(), db.Operations(), kcBuilder, logs),
		LastOperationEndpoint:        broker.NewLastOperation(db.Operations(), db.InstancesArchived(), logs),
		BindEndpoint:                 broker.NewBind(cfg.Broker.Binding, db, logs, clientProvider, kubeconfigProvider, publisher),
		UnbindEndpoint:               broker.NewUnbind(logs, db, brokerBindings.NewServiceAccountBindingsManager(clientProvider, kubeconfigProvider), publisher),
		GetBindingEndpoint:           broker.NewGetBinding(logs, db),
		LastBindingOperationEndpoint: broker.NewLastBindingOperation(logs),
	}

	if r, _ := cfg.GardenerSubscriptionResource(); r == gardener.CredentialsBindingResource {
		kymaEnvBroker.ProvisionEndpoint.UseCredentialsBindings()
		kymaEnvBroker.UpdateEndpoint.UseCredentialsBindings()
	}

	// Wrap broker with panic recovery for all OSB endpoints
	brokerWithPanicRecovery := broker.NewWithPanicRecovery(kymaEnvBroker, logs)

	prefixes := []string{"/{region}", ""}
	subRouter, err := router.NewSubRouter(brokerAPISubrouterName)
	fatalOnError(err, logs)
	broker.AttachRoutes(subRouter, brokerWithPanicRecovery, logs, cfg.Broker.Binding.CreateBindingTimeout, cfg.Broker.DefaultRequestRegion, prefixes)
	router.Handle("/oauth/", http.StripPrefix("/oauth", subRouter))

	respWriter := httputil.NewResponseWriter(logs, cfg.DevelopmentMode)
	runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(db.Instances(), db.Operations(), defaultPlansConfig, cfg.Broker.DefaultRequestRegion, respWriter, publisher, logs)
	router.Handle("/info/runtimes", runtimesInfoHandler)

	// create events endpoint
	eventsHandler := eventshandler.NewHandler(db.Events(), db.Instances())
	router.Handle("/events", eventsHandler)

	versionHandler := version.NewHandler(Version)
	versionHandler.AttachRoutes(router)
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
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("while creating HTTP client for REST mapper: %w", err)
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
	if err != nil {
		err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, false, func(ctx context.Context) (bool, error) {
			mapper, err = apiutil.NewDynamicRESTMapper(cfg, httpClient)
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
