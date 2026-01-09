package setup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/gocraft/dbr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/environmentscleanup"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vrischmann/envconfig"
	"golang.org/x/oauth2/clientcredentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	k8scfg "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type config struct {
	MaxAgeHours   time.Duration `envconfig:"default=24h"`
	LabelSelector string        `envconfig:"default=owner.do-not-delete!=true"`
	Gardener      gardener.Config
	Database      storage.Config
	Broker        broker.ClientConfig
}

type AppBuilder struct {
	cfg            config
	gardenerClient dynamic.ResourceInterface
	db             storage.BrokerStorage
	conn           *dbr.Connection
	brokerClient   *broker.Client
	k8sClient      client.Client
}

type App interface {
	Run() error
}

func NewAppBuilder() AppBuilder {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	return AppBuilder{}
}

func (b *AppBuilder) WithConfig() {
	b.cfg = config{}
	err := envconfig.InitWithPrefix(&b.cfg, "APP")
	if err != nil {
		FatalOnError(fmt.Errorf("while loading cleanup config: %w", err))
	}
}

func (b *AppBuilder) WithGardenerClient() {
	clusterCfg, err := gardener.NewGardenerClusterConfig(b.cfg.Gardener.KubeconfigPath)
	if err != nil {
		FatalOnError(fmt.Errorf("while creating Gardener cluster config: %w", err))
	}
	cli, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		FatalOnError(fmt.Errorf("while creating Gardener client: %w", err))
	}
	gardenerNamespace := fmt.Sprintf("garden-%s", b.cfg.Gardener.Project)
	b.gardenerClient = cli.Resource(gardener.ShootResource).Namespace(gardenerNamespace)
}

func (b *AppBuilder) WithBrokerClient() {
	ctx := context.Background()
	b.brokerClient = broker.NewClient(ctx, b.cfg.Broker)

	clientCfg := clientcredentials.Config{
		ClientID:     b.cfg.Broker.ClientID,
		ClientSecret: b.cfg.Broker.ClientSecret,
		TokenURL:     b.cfg.Broker.TokenURL,
		Scopes:       []string{b.cfg.Broker.Scope},
	}
	httpClientOAuth := clientCfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second
}

func (b *AppBuilder) WithStorage() {
	// Init Storage
	// this job does not write to database, so we do not need to set mode for encryption
	cipher := storage.NewEncrypter(b.cfg.Database.SecretKey)
	var err error
	b.db, b.conn, err = storage.NewFromConfig(b.cfg.Database, events.Config{}, cipher)
	if err != nil {
		FatalOnError(err)
	}
	dbStatsCollector := sqlstats.NewStatsCollector("broker", b.conn)
	prometheus.MustRegister(dbStatsCollector)

}

func (b *AppBuilder) WithK8sClient() {
	err := imv1.AddToScheme(scheme.Scheme)
	FatalOnError(err)
	err = corev1.AddToScheme(scheme.Scheme)
	FatalOnError(err)
	k8sCfg, err := k8scfg.GetConfig()
	FatalOnError(err)
	cli, err := createK8sClient(k8sCfg)
	FatalOnError(err)
	b.k8sClient = cli
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

func (b *AppBuilder) Cleanup() {
	err := b.conn.Close()
	if err != nil {
		FatalOnError(err)
	}

	err = cleaner.HaltIstioSidecar()
	LogOnError(err)

	// do not use defer, close must be done before halting
	err = cleaner.Halt()
	if err != nil {
		FatalOnError(err)
	}
}

func (b *AppBuilder) Create() App {
	return environmentscleanup.NewService(
		b.gardenerClient,
		b.brokerClient,
		b.k8sClient,
		b.db.Instances(),
		b.cfg.MaxAgeHours,
		b.cfg.LabelSelector,
	)
}
