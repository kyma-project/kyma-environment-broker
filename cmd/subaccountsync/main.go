package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/kyma-environment-broker/internal/kymacustomresource"
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})).With("service", "subaccount-sync"))

	slog.Info(fmt.Sprintf("Configuration: event window size:%s, event sync interval:%s, accounts sync interval: %s, storage sync interval: %s, queue sleep interval: %s",
		cfg.EventsWindowSize, cfg.EventsSyncInterval, cfg.AccountsSyncInterval, cfg.StorageSyncInterval, cfg.SyncQueueSleepInterval))

	// create config provider - provider still uses logrus logger
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logrus.WithField("service", "storage"), cfg.KymaVersion),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())

	// create Kyma GVR
	kymaGVR := getResourceKindProvider(cfg.KymaVersion, configProvider)

	// create DB connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, dbConn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher, logrus.WithField("service", "storage"))
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
	syncService := subsync.NewSyncService(AppPrefix, ctx, cfg, kymaGVR, db, dynamicK8sClient)
	syncService.Run()
}

func getK8sClient() client.Client {
	k8sCfg, err := config.GetConfig()
	fatalOnError(err)
	cli, err := createK8sClient(k8sCfg)
	fatalOnError(err)
	return cli
}

func createK8sClient(cfg *rest.Config) (client.Client, error) {
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

func createDynamicK8sClient() dynamic.Interface {
	kcpK8sConfig := config.GetConfigOrDie()
	clusterClient, err := dynamic.NewForConfig(kcpK8sConfig)
	fatalOnError(err)
	return clusterClient
}

func getResourceKindProvider(kymaVersion string, configProvider *kebConfig.ConfigProvider) schema.GroupVersionResource {
	resourceKindProvider := kymacustomresource.NewResourceKindProvider(kymaVersion, configProvider)
	kymaGVR, err := resourceKindProvider.DefaultGvr()
	fatalOnError(err)
	return kymaGVR
}

func fatalOnError(err error) {
	if err != nil {
		panic(err)
	}
}
