package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/cis"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	CIS      cis.Config
	Database storage.Config
	Broker   broker.ClientConfig
}

func main() {
	time.Sleep(20 * time.Second)

	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create logs
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	// create CIS client
	client := cis.NewClient(ctx, cfg.CIS, logger.With("client", "CIS-v2"))

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher)
	fatalOnError(err)
	defer func() { _ = conn.Close() }()

	// create broker client
	brokerClient := broker.NewClient(ctx, cfg.Broker)
	brokerClient.UserAgent = broker.AccountCleanupJob

	// create SubAccountCleanerService and execute process
	sacs := cis.NewSubAccountCleanupService(client, brokerClient, db.Instances(), logger)
	fatalOnError(sacs.Run())
}

func fatalOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
