package provisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
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
	rulesService     *rules.RulesService
}

func NewResolveCredentialsStep(os storage.Operations, accountProvider hyperscaler.AccountProvider, rulesService *rules.RulesService) *ResolveCredentialsStep {
	step := &ResolveCredentialsStep{
		opStorage:       os,
		accountProvider: accountProvider,
		rulesService:    rulesService,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.AccountPoolDependency)
	return step
}

func (s *ResolveCredentialsStep) Name() string {
	return "Resolve_Target_Secret"
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
		msg := fmt.Sprintf("Unable to resolve provisioning secret binding for global account ID %s on Hyperscaler %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey())
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, log)
	}

	log.Info(fmt.Sprintf("Resolved %s as target secret name to use for cluster provisioning for global account ID %s on Hyperscaler %s", targetSecret, operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey()))

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProvisioningParameters.Parameters.TargetSecret = &targetSecret
	}, log)
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
