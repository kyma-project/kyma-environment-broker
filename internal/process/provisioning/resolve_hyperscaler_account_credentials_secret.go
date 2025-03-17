package provisioning

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/client-go/dynamic"
)

type HAPParserResult interface {
	HyperscalerType() string
	IsShared() bool
	IsEUAccess() bool
}

// SecretBinding selectors
const (
	hyperscalerTypeSelectorFmt = gardener.HyperscalerTypeLabelKey + "=%s"
	tenantNameSelectorFmt      = gardener.TenantNameLabelKey + "=%s"

	dirtySelector    = gardener.DirtyLabelKey + "=true"
	internalSelector = gardener.InternalLabelKey + "=true"
	sharedSelector   = gardener.SharedLabelKey + "=true"
	euAccessSelector = gardener.EUAccessLabelKey + "=true"

	notDirtySelector    = `!` + gardener.DirtyLabelKey
	notInternalSelector = `!` + gardener.InternalLabelKey
	notSharedSelector   = `!` + gardener.SharedLabelKey
	notEUAccessSelector = `!` + gardener.EUAccessLabelKey
)

type LabelSelectorBuilder struct {
	strings.Builder
	base string
}

type ResolveHyperscalerAccountCredentialsSecretStep struct {
	operationManager    *process.OperationManager
	secretBindingClient dynamic.ResourceInterface
	opStorage           storage.Operations
	rulesService        *rules.RulesService
}

func NewResolveHyperscalerAccountCredentialsSecretStep(os storage.Operations, secretBindingClient dynamic.ResourceInterface, rulesService *rules.RulesService) *ResolveHyperscalerAccountCredentialsSecretStep {
	step := &ResolveHyperscalerAccountCredentialsSecretStep{
		opStorage:           os,
		secretBindingClient: secretBindingClient,
		rulesService:        rulesService,
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

func (s *ResolveHyperscalerAccountCredentialsSecretStep) createLabelSelectorBuilder(hapParserResult HAPParserResult, tenantName string) *LabelSelectorBuilder {
	b := NewLabelSelectorBuilder()
	b.With(fmt.Sprintf(hyperscalerTypeSelectorFmt, hapParserResult.HyperscalerType()))
	b.With(notDirtySelector)

	if hapParserResult.IsShared() {
		b.With(sharedSelector)
		b.SaveBase()
		return b
	}

	if hapParserResult.IsEUAccess() {
		b.With(euAccessSelector)
	} else {
		b.With(notEUAccessSelector)
	}

	b.SaveBase()
	b.With(fmt.Sprintf(tenantNameSelectorFmt, tenantName))

	return b
}

func NewLabelSelectorBuilder() *LabelSelectorBuilder {
	return &LabelSelectorBuilder{}
}

func (b *LabelSelectorBuilder) With(s string) {
	if b.Len() == 0 {
		b.WriteString(s)
	}
	b.WriteString("," + s)
}

func (b *LabelSelectorBuilder) SaveBase() {
	b.base = b.String()
}

func (b *LabelSelectorBuilder) RevertToBase() {
	b.Reset()
	b.WriteString(b.base)
}
