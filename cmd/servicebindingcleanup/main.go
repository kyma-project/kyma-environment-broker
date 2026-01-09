package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
	"github.com/kyma-project/kyma-environment-broker/internal/servicebindingcleanup"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	Database storage.Config
	Broker   broker.ClientConfig
	Job      JobConfig
}

type JobConfig struct {
	DryRun         bool          `envconfig:"default=true"`
	RequestTimeout time.Duration `envconfig:"default=2s"`
	RequestRetries int           `envconfig:"default=2"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting Service Binding cleanup job")

	var cfg Config
	fatalOnError(envconfig.InitWithPrefix(&cfg, "APP"))

	if cfg.Job.DryRun {
		slog.Info("Dry run only - no changes")
	}

	ctx := context.Background()
	brokerClient := broker.NewClientWithRequestTimeoutAndRetries(ctx, cfg.Broker, cfg.Job.RequestTimeout, cfg.Job.RequestRetries)
	brokerClient.UserAgent = broker.ServiceBindingCleanupJobName

	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher)
	fatalOnError(err)

	svc := servicebindingcleanup.NewService(cfg.Job.DryRun, brokerClient, db.Bindings())
	fatalOnError(svc.PerformCleanup())

	slog.Info("Service Binding cleanup job finished successfully!")

	fatalOnError(conn.Close())
	logOnError(cleaner.HaltIstioSidecar())
	fatalOnError(cleaner.Halt())
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
