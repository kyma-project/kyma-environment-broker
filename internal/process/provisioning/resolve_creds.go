package provisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type ResolveCredentialsStep struct {
	operationManager *process.OperationManager
	accountProvider  hyperscaler.AccountProvider
	opStorage        storage.Operations
	tenant           string
}

func NewResolveCredentialsStep(os storage.Operations, accountProvider hyperscaler.AccountProvider) *ResolveCredentialsStep {
	step := &ResolveCredentialsStep{
		opStorage:       os,
		accountProvider: accountProvider,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.AccountPoolDependency)
	return step
}

func (s *ResolveCredentialsStep) Name() string {
	return "Resolve_Target_Secret"
}

func (s *ResolveCredentialsStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *ResolveCredentialsStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	cloudProvider := operation.InputCreator.Provider()
	effectiveRegion := getEffectiveRegionForSapConvergedCloud(operation.ProvisioningParameters.Parameters.Region)

	hypType, err := hyperscaler.HypTypeFromCloudProviderWithRegion(cloudProvider, &effectiveRegion, &operation.ProvisioningParameters.PlatformRegion)
	if err != nil {
		msg := fmt.Sprintf("failing to determine the type of Hyperscaler to use for planID: %s", operation.ProvisioningParameters.PlanID)
		log.Error(fmt.Sprintf("Aborting after %s", msg))
		return s.operationManager.OperationFailed(operation, msg, err, log)
	}

	euAccess := euaccess.IsEURestrictedAccess(operation.ProvisioningParameters.PlatformRegion)

	log.Info(fmt.Sprintf("HAP lookup for credentials secret binding to provision cluster for global account ID %s on Hyperscaler %s, euAccess %v", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey(), euAccess))

	targetSecret, err := s.getTargetSecretFromGardener(operation, log, hypType, euAccess)
	if err != nil {
		return s.retryOrFailOperation(operation, log, hypType, err)
	}

	operation.ProvisioningParameters.Parameters.TargetSecret = &targetSecret

	updatedOperation, err := s.opStorage.UpdateOperation(operation)
	if err != nil {
		return operation, 1 * time.Minute, nil
	}

	log.Info(fmt.Sprintf("Resolved %s as target secret name to use for cluster provisioning for global account ID %s on Hyperscaler %s", *operation.ProvisioningParameters.Parameters.TargetSecret, operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey()))

	return *updatedOperation, 0, nil
}

func (s *ResolveCredentialsStep) retryOrFailOperation(operation internal.Operation, log *slog.Logger, hypType hyperscaler.Type, err error) (internal.Operation, time.Duration, error) {
	msg := fmt.Sprintf("HAP lookup for secret binding to provision cluster for global account ID %s on Hyperscaler %s has failed", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey())
	errMsg := fmt.Sprintf("%s: %s", msg, err)
	log.Info(errMsg)

	// if failed retry step every 10s by next 10min
	dur := time.Since(operation.UpdatedAt).Round(time.Minute)

	if dur < 10*time.Minute {
		return operation, 10 * time.Second, nil
	}

	log.Error(fmt.Sprintf("Aborting after 10 minutes of failing to resolve provisioning secret binding for global account ID %s on Hyperscaler %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey()))

	return s.operationManager.OperationFailed(operation, msg, err, log)
}

func (s *ResolveCredentialsStep) getTargetSecretFromGardener(operation internal.Operation, log *slog.Logger, hypType hyperscaler.Type, euAccess bool) (string, error) {
	var secretName string
	var err error
	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) || broker.IsSapConvergedCloudPlan(operation.ProvisioningParameters.PlanID) {
		log.Info("HAP lookup for shared secret binding")
		secretName, err = s.accountProvider.GardenerSharedSecretName(hypType, euAccess)
	} else {
		log.Info("HAP lookup for secret binding")
		secretName, err = s.accountProvider.GardenerSecretName(hypType, operation.ProvisioningParameters.ErsContext.GlobalAccountID, euAccess)
	}
	return secretName, err
}

// TODO: Calculate the region parameter using default SapConvergedCloud region. This is to be removed when region is mandatory (Jan 2024).
func getEffectiveRegionForSapConvergedCloud(provisioningParametersRegion *string) string {
	if provisioningParametersRegion != nil && *provisioningParametersRegion != "" {
		return *provisioningParametersRegion
	}
	return provider.DefaultSapConvergedCloudRegion
}
