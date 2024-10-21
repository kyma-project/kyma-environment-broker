package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

type BrokerClient interface {
	Unbind(binding internal.Binding) error
}

type Config struct {
	Database storage.Config
	Broker   broker.ClientConfig
	DryRun   bool `envconfig:"default=true"`
}

type ServiceBindingCleanupService struct {
	cfg             Config
	brokerClient    BrokerClient
	bindingsStorage storage.Bindings
}

func newServiceBindingCleanupService(cfg Config, client BrokerClient, bindingsStorage storage.Bindings) *ServiceBindingCleanupService {
	return &ServiceBindingCleanupService{
		cfg:             cfg,
		brokerClient:    client,
		bindingsStorage: bindingsStorage,
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting Service Binding cleanup job")

	var cfg Config
	fatalOnError(envconfig.InitWithPrefix(&cfg, "APP"))

	if cfg.DryRun {
		slog.Info("Dry run only - no changes")
	}

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher, log.WithField("service", "storage"))
	fatalOnError(err)

	svc := newServiceBindingCleanupService(cfg, brokerClient, db.Bindings())
	fatalOnError(svc.PerformCleanup())

	slog.Info("Service Binding cleanup job finished successfully!")

	fatalOnError(conn.Close())
	logOnError(cleaner.HaltIstioSidecar())
	fatalOnError(cleaner.Halt())
}

func (s *ServiceBindingCleanupService) PerformCleanup() error {
	slog.Info(fmt.Sprintf("Fetching Service Bindings with expires_at <= %q", time.Now().UTC().Truncate(time.Second).String()))
	bindings, err := s.bindingsStorage.ListExpired()
	if err != nil {
		return err
	}

	if s.cfg.DryRun {
		slog.Info(fmt.Sprintf("Expired Service Bindings: %d", len(bindings)))
		return nil
	} else {
		slog.Info("Requesting Service Bindings removal...")
		for _, binding := range bindings {
			err := s.brokerClient.Unbind(binding)
			if err != nil {
				slog.Error(fmt.Sprintf("while sending unbind request for service binding ID %q: %s", binding.ID, err))
				continue
			}
		}
	}
	return nil
}

func fatalOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(0)
	}
}

func logOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
