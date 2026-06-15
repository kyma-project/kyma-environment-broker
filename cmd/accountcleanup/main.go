package main

import (
	"context"
	"fmt"
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
	EventsServiceVersion string `envconfig:"default=v2"`
	CIS                  cis.Config
	Database             storage.Config
	Broker               broker.ClientConfig
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
	var client cis.CisClient
	switch cfg.EventsServiceVersion {
	case "v1":
		log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})).With("client", "CIS-v1")
		client = cis.NewClient(ctx, cfg.CIS, log)
	case "v2":
		log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})).With("client", "CIS-v2")
		client = cis.NewClientV2(ctx, cfg.CIS, log)
	default:
		fatalOnError(fmt.Errorf("Events Service version %s is not supported", cfg.EventsServiceVersion))
	}

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
