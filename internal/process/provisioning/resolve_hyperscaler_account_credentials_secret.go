package provisioning

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type HAPParserResult interface {
	HyperscalerType() string
	IsShared() bool
	IsEUAccess() bool
}

// SecretBinding selectors
const (
	dirtySelector    = gardener.DirtyLabelKey + "=true"
	internalSelector = gardener.InternalLabelKey + "=true"
	sharedSelector   = gardener.SharedLabelKey + "=true"
	euAccessSelector = gardener.EUAccessLabelKey + "=true"

	notDirtySelector    = `!` + gardener.DirtyLabelKey
	notInternalSelector = `!` + gardener.InternalLabelKey
	notSharedSelector   = `!` + gardener.SharedLabelKey
	notEUAccessSelector = `!` + gardener.EUAccessLabelKey
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
	attr := s.provisioningAttributesFromOperationData(operation)

	s.rulesService.Match(attr)

	return operation, 0, errors.New("not implemented")
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) provisioningAttributesFromOperationData(operation internal.Operation) *rules.ProvisioningAttributes {
	return &rules.ProvisioningAttributes{
		Plan:              broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
		PlatformRegion:    operation.ProvisioningParameters.PlatformRegion,
		HyperscalerRegion: operation.Region,
		Hyperscaler:       operation.CloudProvider,
	}
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) buildLabelSelector(hapParserResult HAPParserResult) string {
	var selectorBuilder strings.Builder

	return selectorBuilder.String()
}
