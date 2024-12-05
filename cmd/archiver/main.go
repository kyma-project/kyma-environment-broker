package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/archive"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/vrischmann/envconfig"
)

/**
Archiver is a service which archives the data (already deprovisioned instances) from the database.
It creates a row in instances_archived table and deletes the data from runtime_states and operations tables.
It expects the following environment variables:
- APP_DRY_RUN: if set to true, the service will only log the operations it would perform, without actually performing them
- APP_PERFORM_DELETION: if set to true, the service will perform the deletion of the data from the database
- APP_DATABASE_URL: the URL to the database
- APP_DATABASE_PASSWORD: the password used to connect to the database
- APP_DATABASE_USER: the user used to connect to the database
- APP_DATABASE_NAME: the name of the database
- APP_BATCH_SIZE: the number of instances to process in a single batch
- APP_LOG_LEVEL: the log level for the application, can be: debug, info, warn, error
*/

type Configuration struct {
	DryRun          bool `envconfig:"default=true"`
	PerformDeletion bool `envconfig:"default=false"`
	Database        storage.Config
	BatchSize       int `envconfig:"default=50"`

	LogLevel string `envconfig:"default=info"`
}

func (c Configuration) GetLogLevel() slog.Level {
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

func main() {
	var cfg Configuration
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	logLevel := new(slog.LevelVar)
	logLevel.Set(cfg.GetLogLevel())
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))

	slog.Info(fmt.Sprintf("DryRun: %v", cfg.DryRun))
	slog.Info(fmt.Sprintf("PerformDeletion: %v", cfg.PerformDeletion))
	slog.Info(fmt.Sprintf("Batch size: %v", cfg.BatchSize))

	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, storage.NewEncrypter(cfg.Database.SecretKey))
	fatalOnError(err)
	defer func() {
		err := conn.Close()
		if err != nil {
			slog.Warn(fmt.Sprintf("Failed to close database connection: %s", err.Error()))
		}
	}()

	numberOfInstancesArchived, err := db.InstancesArchived().TotalNumberOfInstancesArchived()
	fatalOnError(err)
	slog.Info(fmt.Sprintf("Total number of instances archived: %d", numberOfInstancesArchived))

	stats, err := db.Instances().DeletedInstancesStatistics()
	fatalOnError(err)
	slog.Info(fmt.Sprintf("Total number of operations for deleted instances: %d", stats.NumberOfOperationsForDeletedInstances))
	slog.Info(fmt.Sprintf("Total number of deleted instances: %d", stats.NumberOfDeletedInstances))

	service := archive.NewService(db, cfg.DryRun, cfg.PerformDeletion, cfg.BatchSize)

	instancesTotal := 0
	operationsTotal := 0
	defer func() {
		slog.Info(fmt.Sprintf("Total number of instances processed: %d", instancesTotal))
		slog.Info(fmt.Sprintf("Total number of operations deleted: %d", operationsTotal))

		stats, _ := db.Instances().DeletedInstancesStatistics()
		slog.Info(fmt.Sprintf("Total number of operations for deleted instances: %d", stats.NumberOfOperationsForDeletedInstances))
		slog.Info(fmt.Sprintf("Total number of deleted instances: %d", stats.NumberOfDeletedInstances))

	}()
	for {
		start := time.Now()
		err, numberOfInstances, numberOfOperations := service.Run()
		elapsed := time.Since(start)
		slog.Info(fmt.Sprintf("%d instances (%d operations) processed in: %v", numberOfInstances, numberOfOperations, elapsed))
		if err != nil {
			slog.Error(fmt.Sprintf("Error during the cleaning process: %s", err.Error()))
			return
		}
		if numberOfInstances == 0 {
			slog.Debug("No more instances to process")
			break
		}
		instancesTotal += numberOfInstances
		operationsTotal += numberOfOperations
		if cfg.DryRun {
			slog.Debug("Dry run: no data was deleted")
			break
		}
	}

}

func fatalOnError(err error) {
	if err != nil {
		panic(err)
	}
}
