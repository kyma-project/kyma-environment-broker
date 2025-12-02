package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/schemamigrator/cleaner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/vrischmann/envconfig"
)

type BrokerClient interface {
	Deprovision(instance internal.Instance) (string, error)
	GetInstanceRequest(instanceID string) (*http.Response, error)
}

type Config struct {
	Database storage.Config
	Broker   broker.ClientConfig
	DryRun   bool `envconfig:"default=true"`
}

type DeprovisionRetriggerService struct {
	cfg             Config
	filter          dbmodel.InstanceFilter
	instanceStorage storage.Instances
	brokerClient    BrokerClient
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting deprovision retrigger job!")

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	if cfg.DryRun {
		slog.Info("Dry run only - no changes")
	}

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey, cfg.Database.Fips.WriteGcm)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher)
	fatalOnError(err)
	svc := newDeprovisionRetriggerService(cfg, brokerClient, db.Instances())

	err = svc.PerformCleanup()

	fatalOnError(err)

	slog.Info("Deprovision retrigger job finished successfully!")
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

func newDeprovisionRetriggerService(cfg Config, brokerClient BrokerClient, instances storage.Instances) *DeprovisionRetriggerService {
	return &DeprovisionRetriggerService{
		cfg:             cfg,
		instanceStorage: instances,
		brokerClient:    brokerClient,
	}
}

func (s *DeprovisionRetriggerService) PerformCleanup() error {
	notCompletelyDeletedFilter := dbmodel.InstanceFilter{DeletionAttempted: &[]bool{true}[0]}
	instancesToDeprovisionAgain, _, _, err := s.instanceStorage.List(notCompletelyDeletedFilter)

	if err != nil {
		slog.Error(fmt.Sprintf("while getting not completely deprovisioned instances: %s", err))
		return err
	}

	if s.cfg.DryRun {
		s.logInstances(instancesToDeprovisionAgain)
		slog.Info(fmt.Sprintf("Instances to retrigger deprovisioning: %d", len(instancesToDeprovisionAgain)))
	} else {
		failuresCount, sanityFailedCount := s.retriggerDeprovisioningForInstances(instancesToDeprovisionAgain)
		deprovisioningAccepted := len(instancesToDeprovisionAgain) - failuresCount - sanityFailedCount
		slog.Info(fmt.Sprintf("Out of %d instances to retrigger deprovisioning: accepted requests = %d, skipped due to sanity failed = %d, failed requests = %d",
			len(instancesToDeprovisionAgain), deprovisioningAccepted, sanityFailedCount, failuresCount))
	}

	return nil
}

func (s *DeprovisionRetriggerService) retriggerDeprovisioningForInstances(instances []internal.Instance) (int, int) {
	var failuresCount int
	var sanityFailedCount int
	for _, instance := range instances {
		// sanity check - if the instance is visible we shall not trigger deprovisioning
		notFound := s.getInstanceReturned404(instance.InstanceID)
		if notFound {
			err := s.deprovisionInstance(instance)
			if err != nil {
				// just counting, logging and ignoring errors
				failuresCount += 1
			}
		} else {
			sanityFailedCount += 1
		}
	}
	return failuresCount, sanityFailedCount
}

func (s *DeprovisionRetriggerService) deprovisionInstance(instance internal.Instance) (err error) {
	slog.Info(fmt.Sprintf("About to deprovision instance for instanceId: %+v", instance.InstanceID))
	operationId, err := s.brokerClient.Deprovision(instance)
	if err != nil {
		slog.Error(fmt.Sprintf("while sending deprovision request for instance ID %s: %s", instance.InstanceID, err))
		return err
	}
	slog.Info(fmt.Sprintf("Deprovision instance for instanceId: %s accepted, operationId: %s", instance.InstanceID, operationId))
	return nil
}

// Sanity check - instance is supposed to be not visible via API. Call should return 404 - NotFound
func (s *DeprovisionRetriggerService) getInstanceReturned404(instanceID string) bool {
	response, err := s.brokerClient.GetInstanceRequest(instanceID)
	if err != nil || response == nil {
		slog.Error(fmt.Sprintf("while trying to GET instance resource for %s: %s", instanceID, err))
		return false
	}
	if response.StatusCode != http.StatusNotFound {
		slog.Error(fmt.Sprintf("unexpectedly GET instance resource for %s: returned %s", instanceID, http.StatusText(response.StatusCode)))
		return false
	}
	return true
}

func (s *DeprovisionRetriggerService) logInstances(instances []internal.Instance) {
	for _, instance := range instances {
		notFound := s.getInstanceReturned404(instance.InstanceID)
		if notFound {
			slog.Info(fmt.Sprintf("instanceId: %s, createdAt: %+v, deletedAt: %+v, GET returned: NotFound", instance.InstanceID, instance.CreatedAt, instance.DeletedAt))
		} else {
			slog.Info(fmt.Sprintf("instanceId: %s, createdAt: %+v, deletedAt: %+v, GET returned: unexpected result", instance.InstanceID, instance.CreatedAt, instance.DeletedAt))
		}
	}
}

func fatalOnError(err error) {
	if err != nil {
		// exit with 0 to avoid any side effects - we ignore all errors only logging those
		slog.Error(err.Error())
		os.Exit(0)
	}
}

func logOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}
