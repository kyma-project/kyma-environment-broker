package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/kymacustomresource"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/vrischmann/envconfig"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	subsync "github.com/kyma-project/kyma-environment-broker/internal/subaccountsync"
)

const AppPrefix = "subaccount_sync"

func main() {
	// create context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cli := getK8sClient()

	// create and fill config
	var cfg subsync.Config
	err := envconfig.InitWithPrefix(&cfg, AppPrefix)
	if err != nil {
		fatalOnError(err)
	}

	logLevel := new(slog.LevelVar)
	logLevel.Set(cfg.GetLogLevel())
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})).With("service", "subaccount-sync")
	slog.SetDefault(logger)

	slog.Info(fmt.Sprintf("Configuration: events window size:%s, events sync interval:%s, accounts sync interval: %s, storage sync interval: %s, queue sleep interval: %s",
		cfg.EventsWindowSize, cfg.EventsWindowInterval, cfg.AccountsSyncInterval, cfg.StorageSyncInterval, cfg.SyncQueueSleepInterval))
	slog.Info(fmt.Sprintf("Configuration: updateResources: %t", cfg.UpdateResources))
	slog.Info(fmt.Sprintf("Configuration: alwaysSubaccountFromDatabase: %t", cfg.AlwaysSubaccountFromDatabase))

	if cfg.EventsWindowSize < cfg.EventsWindowInterval {
		slog.Warn("Events window size is smaller than events sync interval. This might cause missing events so we set window size to the interval.")
		cfg.EventsWindowSize = cfg.EventsWindowInterval
	}

	// create config provider
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logger.With("component", "config-map-reader")),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())

	// create Kyma GVR
	kymaGVR := getResourceKindProvider(kebConfig.NewConfigMapConfigProvider(configProvider, cfg.RuntimeConfigurationConfigMapName, kebConfig.RuntimeConfigurationRequiredFields))

	// create DB connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, dbConn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher)

	// create and register metrics
	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(collectors.NewGoCollector())

	dbStatsCollector := sqlstats.NewStatsCollector("broker", dbConn)
	metricsRegistry.MustRegister(dbStatsCollector)

	if err != nil {
		fatalOnError(err)
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic. Error:\n", r)
		}
		err = dbConn.Close()
		if err != nil {
			slog.Warn(fmt.Sprintf("failed to close database connection: %s", err.Error()))
		}
	}()

	// create dynamic K8s client
	dynamicK8sClient := createDynamicK8sClient()

	// create service
	syncService := subsync.NewSyncService(AppPrefix, ctx, cfg, kymaGVR, db, dynamicK8sClient, metricsRegistry)
	syncService.Run()
}

func getK8sClient() client.Client {
	k8sCfg, err := config.GetConfig()
	fatalOnError(err)

	// Configure TLS for FIPS compliance
	configureTLSForFIPS(k8sCfg)

	cli, err := createK8sClient(k8sCfg)
	fatalOnError(err)
	return cli
}

func configureTLSForFIPS(cfg *rest.Config) {
	if cfg.TLSClientConfig.CAData == nil && cfg.TLSClientConfig.CAFile == "" {
		// Use the default CA certificate bundle if none is specified
		cfg.TLSClientConfig.CAFile = "/etc/ssl/certs/ca-certificates.crt"
	}

	// For FIPS compliance, ensure we use secure TLS configuration
	if cfg.Transport == nil || cfg.WrapTransport == nil {
		cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			if transport, ok := rt.(*http.Transport); ok {
				if transport.TLSClientConfig == nil {
					transport.TLSClientConfig = &tls.Config{}
				}
				// Ensure minimum TLS version for FIPS compliance
				transport.TLSClientConfig.MinVersion = tls.VersionTLS12
				// Use only FIPS-approved cipher suites
				transport.TLSClientConfig.CipherSuites = []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				}
			}
			return rt
		}
	}
}

func createK8sClient(cfg *rest.Config) (client.Client, error) {
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

func createDynamicK8sClient() dynamic.Interface {
	kcpK8sConfig := config.GetConfigOrDie()

	// Configure TLS for FIPS compliance
	configureTLSForFIPS(kcpK8sConfig)

	clusterClient, err := dynamic.NewForConfig(kcpK8sConfig)
	fatalOnError(err)
	return clusterClient
}

func getResourceKindProvider(configProvider kebConfig.ConfigMapConfigProvider) schema.GroupVersionResource {
	resourceKindProvider := kymacustomresource.NewResourceKindProvider(configProvider)
	kymaGVR, err := resourceKindProvider.DefaultGvr()
	fatalOnError(err)
	return kymaGVR
}

func fatalOnError(err error) {
	if err != nil {
		panic(err)
	}
}
