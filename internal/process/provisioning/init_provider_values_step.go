package provisioning

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

/*
InitProviderValuesStep step is responsible to create values specific to the plan and store it in the DB (as a field in the operation) for time of provisioning.
Those values can be used across the provisioning process and won't be used later - they are not persisted in the InstanceDetails object.
*/
type InitProviderValuesStep struct {
	operationManager           *process.OperationManager
	config                     broker.InfrastructureManager
	trialPlatformRegionMapping map[string]string

	instanceStorage storage.Instances
}

var _ process.Step = &InitProviderValuesStep{}

func NewInitProviderValuesStep(os storage.Operations, is storage.Instances, infrastructureManagerConfig broker.InfrastructureManager, trialPlatformRegionMapping map[string]string) *InitProviderValuesStep {
	return &InitProviderValuesStep{
		operationManager:           process.NewOperationManager(os, "InitProviderValuesStep", kebError.KEBDependency),
		config:                     infrastructureManagerConfig,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		instanceStorage:            is,
	}
}

func (s InitProviderValuesStep) Name() string {
	return "Init_Provider_Values"
}

func (s *InitProviderValuesStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.ProviderValues != nil {
		logger.Info("Provider values already exist, skipping step")
		return operation, 0, nil
	}
	values, err := provider.GetPlanSpecificValues(&operation, s.config.MultiZoneCluster, s.config.DefaultTrialProvider, s.config.UseSmallerMachineTypes, s.trialPlatformRegionMapping,
		s.config.DefaultGardenerShootPurpose, s.config.ControlPlaneFailureTolerance)
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while calculating plan specific values : %s", err), err, logger)
	}

	err = s.updateInstance(operation.InstanceID, runtime.CloudProviderFromString(values.ProviderType))
	if err != nil {
		return s.operationManager.RetryOperation(operation, fmt.Sprintf("while updating instance with provider: %s", err.Error()), err, kcpRetryInterval, kcpRetryTimeout, logger)
	}

	logger.Debug(fmt.Sprintf("Saving plan specific values: provider=%s, region=%s, purpose=%s, failureTolerance=%v", values.ProviderType, values.Region, values.Purpose, values.FailureTolerance))

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProviderValues = &values
		op.CloudProvider = string(runtime.CloudProviderFromString(values.ProviderType))
	}, logger)
}

func (s *InitProviderValuesStep) updateInstance(id string, provider runtime.CloudProvider) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return fmt.Errorf("while getting instance: %w", err)
	}
	instance.Provider = provider
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return fmt.Errorf("while updating instance: %w", err)
	}

	return nil
}
