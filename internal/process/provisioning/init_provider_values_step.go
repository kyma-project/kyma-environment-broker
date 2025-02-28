package provisioning

import (
	"fmt"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"log/slog"
	"time"
)

/*
InitProviderValuesStep step is responsible to create values specific to the plan and store it in the DB (as a field in the operation) for time of provisioning.
Those values can be used across the provisioning process and won't be used later - they are not persisted in the InstanceDetails object.
*/
type InitProviderValuesStep struct {
	operationManager           *process.OperationManager
	config                     input.Config
	trialPlatformRegionMapping map[string]string
	useSmallerMachineTypes     bool
}

var _ process.Step = &InitProviderValuesStep{}

func NewInitProviderValuesStep(os storage.Operations, cfg input.Config, trialPlatformRegionMapping map[string]string, useSmallerMachineTypes bool) *InitProviderValuesStep {
	return &InitProviderValuesStep{
		operationManager:           process.NewOperationManager(os, "InitProviderValuesStep", kebError.KEBDependency),
		config:                     cfg,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		useSmallerMachineTypes:     useSmallerMachineTypes,
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
	values, err := provider.GetPlanSpecificValues(&operation, s.config.MultiZoneCluster, s.config.DefaultTrialProvider, s.useSmallerMachineTypes, s.trialPlatformRegionMapping,
		s.config.DefaultGardenerShootPurpose, s.config.ControlPlaneFailureTolerance)
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while calculating plan specific values : %s", err), err, logger)
	}

	logger.Debug(fmt.Sprintf("Saving plan specific values: provider=%s, region=%s, purpose=%s, failureTolerance=%v", values.ProviderType, values.Region, values.Purpose, values.FailureTolerance))

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProviderValues = &values
		op.CloudProvider = string(runtime.CloudProviderFromString(values.ProviderType))
	}, logger)
}
