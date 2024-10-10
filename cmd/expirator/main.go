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
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

type BrokerClient interface {
	SendExpirationRequest(instance internal.Instance) (bool, error)
}

type instancePredicate func(internal.Instance) bool

type Config struct {
	Database         storage.Config
	Broker           broker.ClientConfig
	DryRun           bool          `envconfig:"default=true"`
	ExpirationPeriod time.Duration `envconfig:"default=720h"` // 30 days
	TestRun          bool          `envconfig:"default=false"`
	TestSubaccountID string        `envconfig:"default=prow-keb-trial-suspension"`
	PlanID           string        `envconfig:"default=7d55d31d-35ae-4438-bf13-6ffdfa107d9f"`
}

type CleanupService struct {
	cfg             Config
	filter          dbmodel.InstanceFilter
	instanceStorage storage.Instances
	brokerClient    BrokerClient
	planID          string
}

func newCleanupService(cfg Config, brokerClient BrokerClient, instances storage.Instances) *CleanupService {
	return &CleanupService{
		cfg:             cfg,
		instanceStorage: instances,
		brokerClient:    brokerClient,
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting cleanup job")

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	if cfg.DryRun {
		slog.Info("Dry run only - no changes")
	}

	slog.Info(fmt.Sprintf("Expiration period: %+v", cfg.ExpirationPeriod))
	slog.Info(fmt.Sprintf("PlanID: %s", cfg.PlanID))

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher, log.WithField("service", "storage"))
	fatalOnError(err)
	svc := newCleanupService(cfg, brokerClient, db.Instances())

	err = svc.PerformCleanup()

	fatalOnError(err)

	slog.Info("Expirator job finished successfully!")

	err = conn.Close()
	if err != nil {
		fatalOnError(err)
	}

	err = cleaner.HaltIstioSidecar()
	logOnError(err)
	// do not use defer, close must be done before halting
	err = cleaner.Halt()
	fatalOnError(err)
}

func (s *CleanupService) PerformCleanup() error {

	filter := dbmodel.InstanceFilter{PlanIDs: []string{s.cfg.PlanID}}
	if s.cfg.TestRun {
		filter.SubAccountIDs = []string{s.cfg.TestSubaccountID}
	}
	instances, count, err := s.getInstances(filter)

	if err != nil {
		slog.Error(fmt.Sprintf("while getting instances: %s", err))
		return err
	}

	instancesToExpire, instancesToExpireCount := s.filterInstances(
		instances,
		func(instance internal.Instance) bool { return time.Since(instance.CreatedAt) >= s.cfg.ExpirationPeriod },
	)

	instancesToBeLeftCount := count - instancesToExpireCount

	if s.cfg.DryRun {
		s.logInstances(instancesToExpire)
		slog.Info(fmt.Sprintf("Instances: %+v, to expire now: %+v, to be left non-expired: %+v", count, instancesToExpireCount, instancesToBeLeftCount))
	} else {
		suspensionsAcceptedCount, onlyMarkedAsExpiredCount, failuresCount := s.cleanupInstances(instancesToExpire)
		slog.Info(fmt.Sprintf("Instances: %+v, to expire: %+v, left non-expired: %+v, suspension under way: %+v just marked expired: %+v, failures: %+v",
			count, instancesToExpireCount, instancesToBeLeftCount, suspensionsAcceptedCount, onlyMarkedAsExpiredCount, failuresCount))
	}
	return nil
}

func (s *CleanupService) getInstances(filter dbmodel.InstanceFilter) ([]internal.Instance, int, error) {

	instances, _, totalCount, err := s.instanceStorage.List(filter)
	if err != nil {
		return []internal.Instance{}, 0, err
	}

	return instances, totalCount, nil
}

func (s *CleanupService) filterInstances(instances []internal.Instance, filter instancePredicate) ([]internal.Instance, int) {
	var filteredInstances []internal.Instance
	for _, instance := range instances {
		if filter(instance) {
			filteredInstances = append(filteredInstances, instance)
		}
	}
	return filteredInstances, len(filteredInstances)
}

func (s *CleanupService) cleanupInstances(instances []internal.Instance) (int, int, int) {
	var suspensionAccepted int
	var onlyExpirationMarked int
	totalInstances := len(instances)
	for _, instance := range instances {
		suspensionUnderWay, err := s.expireInstance(instance)
		if err != nil {
			// ignoring errors - only logging
			slog.Error(fmt.Sprintf("while sending expiration request for instanceID: %s, error: %s", instance.InstanceID, err))
			continue
		}
		if suspensionUnderWay {
			suspensionAccepted += 1
		} else {
			onlyExpirationMarked += 1
		}
	}
	failures := totalInstances - suspensionAccepted - onlyExpirationMarked
	return suspensionAccepted, onlyExpirationMarked, failures
}

func (s *CleanupService) logInstances(instances []internal.Instance) {
	for _, instance := range instances {
		slog.Info(fmt.Sprintf("instanceId: %+v createdAt: %+v (%.0f days ago) servicePlanID: %+v servicePlanName: %+v",
			instance.InstanceID, instance.CreatedAt, time.Since(instance.CreatedAt).Hours()/24, instance.ServicePlanID, instance.ServicePlanName))
	}
}

func (s *CleanupService) expireInstance(instance internal.Instance) (processed bool, err error) {
	slog.Info(fmt.Sprintf("About to make instance expired for instanceID: %+v", instance.InstanceID))
	suspensionUnderWay, err := s.brokerClient.SendExpirationRequest(instance)
	if err != nil {
		slog.Error(fmt.Sprintf("while sending expiration request for instanceID %q: %s", instance.InstanceID, err))
		return suspensionUnderWay, err
	}
	return suspensionUnderWay, nil
}

func fatalOnError(err error) {
	if err != nil {
		// temporarily we exit with 0 to avoid any side effects - we ignore all errors only logging those
		//log.Fatal(err)
		slog.Error(err.Error())
		os.Exit(0)
	}
}

func logOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
