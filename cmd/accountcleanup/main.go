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
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	ClientVersion string
	CIS           cis.Config
	Database      storage.Config
	Broker        broker.ClientConfig
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
	switch cfg.ClientVersion {
	case "v1.0":
		fatalOnError(fmt.Errorf("CIS V1 is not supported anymore, please use V2"))
	case "v2.0":
		log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})).With("client", "CIS-2.0")
		client = cis.NewClient(ctx, cfg.CIS, log)
	default:
		fatalOnError(fmt.Errorf("client version %s is not supported", cfg.ClientVersion))
	}

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey, cfg.Database.Fips.WriteGcm)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher)
	fatalOnError(err)

	// create broker client
	brokerClient := broker.NewClient(ctx, cfg.Broker)
	brokerClient.UserAgent = broker.AccountCleanupJob

	// create SubAccountCleanerService and execute process
	sacs := cis.NewSubAccountCleanupService(client, brokerClient, db.Instances())
	fatalOnError(sacs.Run())

	// do not use defer, close must be done before halting
	err = conn.Close()
	if err != nil {
		fatalOnError(err)
	}

	err = cleaner.HaltIstioSidecar()
	logOnError(err)
	err = cleaner.Halt()
	fatalOnError(err)

	time.Sleep(5 * time.Second)
}

func fatalOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func logOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
