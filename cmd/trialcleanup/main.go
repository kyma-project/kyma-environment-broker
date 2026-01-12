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
	"github.com/vrischmann/envconfig"
)

const (
	trialPlanID = broker.TrialPlanID
)

type BrokerClient interface {
	SendExpirationRequest(instance internal.Instance) (bool, error)
}

type Config struct {
	Database         storage.Config
	Broker           broker.ClientConfig
	DryRun           bool          `envconfig:"default=true"`
	ExpirationPeriod time.Duration `envconfig:"default=336h"`
	TestRun          bool          `envconfig:"default=false"`
	TestSubaccountID string        `envconfig:"default=prow-keb-trial-suspension"`
}

type TrialCleanupService struct {
	cfg             Config
	filter          dbmodel.InstanceFilter
	instanceStorage storage.Instances
	brokerClient    BrokerClient
}

type instancePredicate func(internal.Instance) bool

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting trial cleanup job")

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	if cfg.DryRun {
		slog.Info("Dry run only - no changes")
	}

	slog.Info(fmt.Sprintf("Expiration period: %+v", cfg.ExpirationPeriod))

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher)
	fatalOnError(err)
	svc := newTrialCleanupService(cfg, brokerClient, db.Instances())

	err = svc.PerformCleanup()

	fatalOnError(err)

	slog.Info("Trial cleanup job finished successfully!")

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

func newTrialCleanupService(cfg Config, brokerClient BrokerClient, instances storage.Instances) *TrialCleanupService {
	return &TrialCleanupService{
		cfg:             cfg,
		instanceStorage: instances,
		brokerClient:    brokerClient,
	}
}

func (s *TrialCleanupService) PerformCleanup() error {

	trialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{trialPlanID}}
	if s.cfg.TestRun {
		trialInstancesFilter.SubAccountIDs = []string{s.cfg.TestSubaccountID}
	}
	trialInstances, trialInstancesCount, err := s.getInstances(trialInstancesFilter)

	if err != nil {
		slog.Error(fmt.Sprintf("while getting trial instances: %s", err))
		return err
	}

	instancesToExpire, instancesToExpireCount := s.filterInstances(
		trialInstances,
		func(instance internal.Instance) bool { return time.Since(instance.CreatedAt) >= s.cfg.ExpirationPeriod },
	)

	instancesToBeLeftCount := trialInstancesCount - instancesToExpireCount

	if s.cfg.DryRun {
		s.logInstances(instancesToExpire)
		slog.Info(fmt.Sprintf("Trials: %+v, to expire now: %+v, to be left non-expired: %+v", trialInstancesCount, instancesToExpireCount, instancesToBeLeftCount))
	} else {
		suspensionsAcceptedCount, onlyMarkedAsExpiredCount, failuresCount := s.cleanupInstances(instancesToExpire)
		slog.Info(fmt.Sprintf("Trials: %+v, to expire: %+v, left non-expired: %+v, suspension under way: %+v, just marked expired: %+v, failures: %+v",
			trialInstancesCount, instancesToExpireCount, instancesToBeLeftCount, suspensionsAcceptedCount, onlyMarkedAsExpiredCount, failuresCount))
	}
	return nil
}

func (s *TrialCleanupService) getInstances(filter dbmodel.InstanceFilter) ([]internal.Instance, int, error) {

	instances, _, totalCount, err := s.instanceStorage.List(filter)
	if err != nil {
		return []internal.Instance{}, 0, err
	}

	return instances, totalCount, nil
}

func (s *TrialCleanupService) filterInstances(instances []internal.Instance, filter instancePredicate) ([]internal.Instance, int) {
	var filteredInstances []internal.Instance
	for _, instance := range instances {
		if filter(instance) {
			filteredInstances = append(filteredInstances, instance)
		}
	}
	return filteredInstances, len(filteredInstances)
}

func (s *TrialCleanupService) cleanupInstances(instances []internal.Instance) (int, int, int) {
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

func (s *TrialCleanupService) logInstances(instances []internal.Instance) {
	for _, instance := range instances {
		slog.Info(fmt.Sprintf("instanceId: %+v createdAt: %+v (%.0f days ago) servicePlanID: %+v servicePlanName: %+v",
			instance.InstanceID, instance.CreatedAt, time.Since(instance.CreatedAt).Hours()/24, instance.ServicePlanID, instance.ServicePlanName))
	}
}

func (s *TrialCleanupService) expireInstance(instance internal.Instance) (processed bool, err error) {
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
