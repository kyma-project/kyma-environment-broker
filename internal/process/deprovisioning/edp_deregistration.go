package deprovisioning

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

//go:generate mockery --name=EDPClient --output=automock --outpkg=automock --case=underscore
type EDPClient interface {
	DeleteDataTenant(name, env string, log *slog.Logger) error
	DeleteMetadataTenant(name, env, key string, log *slog.Logger) error
}

type EDPDeregistrationStep struct {
	operationManager *process.OperationManager
	client           EDPClient
	config           edp.Config
	dbInstances      storage.Instances
	dbOperations     storage.Operations
}

type InstanceOperationStorage interface {
	storage.Operations
	storage.Instances
}

func NewEDPDeregistrationStep(os storage.Operations, is storage.Instances, client EDPClient, config edp.Config) *EDPDeregistrationStep {
	step := &EDPDeregistrationStep{
		client:       client,
		config:       config,
		dbOperations: os,
		dbInstances:  is,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.EDPDependency)
	return step
}

func (s *EDPDeregistrationStep) Name() string {
	return "EDP_Deregistration"
}

func (s *EDPDeregistrationStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	instances, err := s.dbInstances.FindAllInstancesForSubAccounts([]string{operation.SubAccountID})
	if err != nil {
		log.Error(fmt.Sprintf("Unable to get instances for given subaccount: %s", err.Error()))
		return operation, time.Second, nil
	}
	// check if there is any other instance for given subaccount and such instances are not being deprovisioned
	numberOfInstancesWithEDP := 0
	var edpInstanceIDs []string
	for _, instance := range instances {
		lastOperation, err := s.dbOperations.GetLastOperation(instance.InstanceID)
		if err != nil {
			log.Error(fmt.Sprintf("Unable to get last operation for given instance (Id=%s): %s", instance.InstanceID, err.Error()))
			return operation, time.Second, nil
		}
		if lastOperation.Type != internal.OperationTypeDeprovision {
			numberOfInstancesWithEDP = numberOfInstancesWithEDP + 1
			edpInstanceIDs = append(edpInstanceIDs, operation.InstanceID)
		}
	}
	if numberOfInstancesWithEDP > 0 {
		log.Info(fmt.Sprintf("Skipping EDP deregistration due to existing other instances: %s", strings.Join(edpInstanceIDs, ", ")))
		return operation, 0, nil
	}

	log.Info("Delete DataTenant metadata")

	subAccountID := strings.ToLower(operation.SubAccountID)
	for _, key := range []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	} {
		err := s.client.DeleteMetadataTenant(subAccountID, s.config.Environment, key, log)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("cannot remove DataTenant metadata with key: %s", key))
		}
	}

	log.Info("Delete DataTenant")
	err = s.client.DeleteDataTenant(subAccountID, s.config.Environment, log)
	if err != nil {
		return s.handleError(operation, err, log, "cannot remove DataTenant")
	}

	return operation, 0, nil
}

func (s *EDPDeregistrationStep) handleError(operation internal.Operation, err error, log *slog.Logger, msg string) (internal.Operation, time.Duration, error) {
	if kebError.IsTemporaryError(err) {
		since := time.Since(operation.UpdatedAt)
		if since < time.Minute*30 {
			log.Warn(fmt.Sprintf("request to EDP failed: %s. Retry...", err))
			return operation, 10 * time.Second, nil
		}
	}

	errMsg := fmt.Sprintf("Step %s failed. EDP data have not been deleted.", s.Name())
	operation, repeat, err := s.operationManager.MarkStepAsExcutedButNotCompleted(operation, s.Name(), errMsg, log)
	if repeat != 0 {
		return operation, repeat, err
	}

	log.Error(fmt.Sprintf("%s: %s", msg, err))

	return operation, 0, nil
}
