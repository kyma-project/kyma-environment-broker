package main

import (
	"context"
	"time"

	btpmanager "github.com/kyma-project/kyma-environment-broker/internal/btpmanager/credentials"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Config struct {
	Database               storage.Config
	Events                 events.Config
	DryRun                 bool   `envconfig:"default=true"`
	JobEnabled             bool   `envconfig:"default=false"`
	JobInterval            int    `envconfig:"default=24"`
	JobReconciliationDelay string `envconfig:"default=0s"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	logs.Info("runtime-reconciler started")
	logs.Info("runtime-reconciler debug version: 1")

	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "RUNTIME_RECONCILER")
	fatalOnError(err, logs)
	logs.Info("runtime-reconciler config loaded")
	if !cfg.JobEnabled {
		logs.Info("job disabled, module stopped.")
		return
	}
	jobReconciliationDelay, err := time.ParseDuration(cfg.JobReconciliationDelay)
	if cfg.JobEnabled && err != nil {
		fatalOnError(err, logs)
	}

	logs.Infof("runtime-listener runing as dry run? %t", cfg.DryRun)

	cipher := storage.NewEncrypter(cfg.Database.SecretKey)

	db, _, err := storage.NewFromConfig(cfg.Database, cfg.Events, cipher, logs.WithField("service", "storage"))
	fatalOnError(err, logs)
	logs.Info("runtime-reconciler connected to database")

	kcpK8sConfig, err := config.GetConfig()
	fatalOnError(err, logs)
	kcpK8sClient, err := client.New(kcpK8sConfig, client.Options{})
	fatalOnError(err, logs)

	btpOperatorManager := btpmanager.NewManager(ctx, kcpK8sClient, db.Instances(), logs, cfg.DryRun)

	logs.Infof("job enabled? %t", cfg.JobEnabled)
	if cfg.JobEnabled {
		btpManagerCredentialsJob := btpmanager.NewJob(btpOperatorManager, logs)
		logs.Infof("runtime-reconciler created job every %d m", cfg.JobInterval)
		btpManagerCredentialsJob.Start(cfg.JobInterval, jobReconciliationDelay)
	}

	<-ctx.Done()
}

func fatalOnError(err error, log *logrus.Logger) {
	if err != nil {
		log.Fatal(err)
	}
}
