package provisioning

import (
	"errors"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal"
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
	return operation, 0, errors.New("not implemented")
}
