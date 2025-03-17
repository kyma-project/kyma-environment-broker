package provisioning

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type ResolveHyperscalerAccountCredentialsSecretStep struct {
	operationManager *process.OperationManager
	accountProvider  hyperscaler.AccountProvider
	opStorage        storage.Operations
	rulesService     *rules.RulesService
}

func NewResolveHyperscalerAccountCredentialsSecretStep(os storage.Operations, accountProvider hyperscaler.AccountProvider, rulesService *rules.RulesService) *ResolveHyperscalerAccountCredentialsSecretStep {
	step := &ResolveHyperscalerAccountCredentialsSecretStep{
		opStorage:       os,
		accountProvider: accountProvider,
		rulesService:    rulesService,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.AccountPoolDependency)
	return step
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) Name() string {
	return "Resolve_Hyperscaler_Account_Credentials_Secret"
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	attr, err := s.provisioningAttributesFromOperationData(operation)
	if err != nil {
		msg := fmt.Sprintf("%s for %s plan", err, broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID])
		return s.operationManager.OperationFailed(operation, msg, err, log)
	}

	s.rulesService.Match(attr)

	return operation, 0, errors.New("not implemented")
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) provisioningAttributesFromOperationData(operation internal.Operation) (*rules.ProvisioningAttributes, error) {
	cloudProvider := runtime.CloudProviderFromString(operation.ProviderValues.ProviderType)
	effectiveRegion := getEffectiveRegionForSapConvergedCloud(operation.ProvisioningParameters.Parameters.Region)
	hypType, err := hyperscaler.HypTypeFromCloudProviderWithRegion(cloudProvider, &effectiveRegion, &operation.ProvisioningParameters.PlatformRegion)
	if err != nil {
		return nil, err
	}

	return &rules.ProvisioningAttributes{
		Plan:              broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
		PlatformRegion:    operation.ProvisioningParameters.PlatformRegion,
		HyperscalerRegion: hypType.GetRegion(),
		Hyperscaler:       hypType.GetKey(),
	}, nil
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) buildLabelSelector(labels map[string]string) string {
	return ""
}
