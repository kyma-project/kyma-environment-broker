package provisioning

import (
	"context"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type HAPParserResult interface {
	HyperscalerType() string
	IsShared() bool
	IsEUAccess() bool
}

const requestTimeout = 10 * time.Second

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
	operationManager *process.OperationManager
	gardenerClient   *gardener.Client
	opStorage        storage.Operations
	rulesService     *rules.RulesService
}

func NewResolveHyperscalerAccountCredentialsSecretStep(os storage.Operations, gardenerClient *gardener.Client, rulesService *rules.RulesService) *ResolveHyperscalerAccountCredentialsSecretStep {
	step := &ResolveHyperscalerAccountCredentialsSecretStep{
		opStorage:      os,
		gardenerClient: gardenerClient,
		rulesService:   rulesService,
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
	var hapParserResult HAPParserResult
	labelSelectorBuilder := s.createLabelSelectorBuilder(hapParserResult, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	secretBindings, err := s.getSecretBindings(labelSelectorBuilder.String())
	_ = secretBindings
	if err != nil {
		msg := fmt.Sprintf("listing service bindings with selector %q", labelSelectorBuilder.String())
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, log)
	}

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

func (s *ResolveHyperscalerAccountCredentialsSecretStep) getSecretBindings(labelSelector string) (*unstructured.UnstructuredList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return s.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(s.gardenerClient.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
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
