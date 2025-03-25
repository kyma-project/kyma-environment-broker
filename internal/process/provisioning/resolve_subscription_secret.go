package provisioning

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type ParsedRule interface {
	Hyperscaler() string
	IsShared() bool
	IsEUAccess() bool
	Rule() string
}

// SecretBinding selector requirements
const (
	hyperscalerTypeReqFmt = gardener.HyperscalerTypeLabelKey + "=%s"
	tenantNameReqFmt      = gardener.TenantNameLabelKey + "=%s"

	dirtyReq    = gardener.DirtyLabelKey + "=true"
	internalReq = gardener.InternalLabelKey + "=true"
	sharedReq   = gardener.SharedLabelKey + "=true"
	euAccessReq = gardener.EUAccessLabelKey + "=true"

	notSharedReq = gardener.SharedLabelKey + `!=true`

	notDirtyReq       = `!` + gardener.DirtyLabelKey
	notInternalReq    = `!` + gardener.InternalLabelKey
	notEUAccessReq    = `!` + gardener.EUAccessLabelKey
	notTenantNamedReq = `!` + gardener.TenantNameLabelKey
)

type LabelSelectorBuilder struct {
	strings.Builder
	base string
}

type ResolveSubscriptionSecretStep struct {
	operationManager *process.OperationManager
	gardenerClient   *gardener.Client
	opStorage        storage.Operations
	rulesService     *rules.RulesService
	stepRetryTuple   internal.RetryTuple
	mu               sync.Mutex
}

func NewResolveSubscriptionSecretStep(os storage.Operations, gardenerClient *gardener.Client, rulesService *rules.RulesService, stepRetryTuple internal.RetryTuple) *ResolveSubscriptionSecretStep {
	step := &ResolveSubscriptionSecretStep{
		opStorage:      os,
		gardenerClient: gardenerClient,
		rulesService:   rulesService,
		stepRetryTuple: stepRetryTuple,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.AccountPoolDependency)
	return step
}

func (s *ResolveSubscriptionSecretStep) Name() string {
	return "Resolve_Subscription_Secret"
}

func (s *ResolveSubscriptionSecretStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	targetSecretName, err := s.resolveSecretName(operation, log)
	if err != nil {
		msg := fmt.Sprintf("resolving secret name")
		return s.operationManager.RetryOperation(operation, msg, err, s.stepRetryTuple.Interval, s.stepRetryTuple.Timeout, log)
	}

	if targetSecretName == "" {
		return s.operationManager.OperationFailed(operation, "failed to determine secret name", fmt.Errorf("target secret name is empty"), log)
	}

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProvisioningParameters.Parameters.TargetSecret = &targetSecretName
	}, log)
}

func (s *ResolveSubscriptionSecretStep) resolveSecretName(operation internal.Operation, log *slog.Logger) (string, error) {
	attr := s.provisioningAttributesFromOperationData(operation)

	log.Info(fmt.Sprintf("matching provisioning attributes %q to filtering rule", attr))
	parsedRule, err := s.matchProvisioningAttributesToRule(attr)
	if err != nil {
		return "", err
	}

	log.Info(fmt.Sprintf("matched rule: %q", parsedRule.Rule()))
	labelSelectorBuilder := s.createLabelSelectorBuilder(parsedRule, operation.ProvisioningParameters.ErsContext.GlobalAccountID)

	log.Info(fmt.Sprintf("getting secret binding with selector %q", labelSelectorBuilder.String()))
	if parsedRule.IsShared() {
		return s.getSharedSecretName(labelSelectorBuilder.String())
	}

	secretBinding, err := s.getSecretBinding(labelSelectorBuilder.String())
	if err != nil && !kebError.IsNotFoundError(err) {
		return "", err
	}

	if secretBinding != nil {
		return secretBinding.GetSecretRefName(), nil
	}

	log.Info(fmt.Sprintf("no secret binding found for tenant: %q", operation.ProvisioningParameters.ErsContext.GlobalAccountID))

	s.mu.Lock()
	defer s.mu.Unlock()

	labelSelectorBuilder.RevertToBase()
	labelSelectorBuilder.ForSecretBindingClaim()

	log.Info(fmt.Sprintf("getting secret binding with selector %q", labelSelectorBuilder.String()))
	secretBinding, err = s.getSecretBinding(labelSelectorBuilder.String())
	if err != nil {
		if kebError.IsNotFoundError(err) {
			return "", fmt.Errorf("failed to find unassigned secret binding with selector %q", labelSelectorBuilder.String())
		}
		return "", err
	}

	log.Info(fmt.Sprintf("claiming secret binding for tenant %q", operation.ProvisioningParameters.ErsContext.GlobalAccountID))
	secretBinding, err = s.claimSecretBinding(secretBinding, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	if err != nil {
		return "", fmt.Errorf("while claiming secret binding for tenant: %s: %w", operation.ProvisioningParameters.ErsContext.GlobalAccountID, err)
	}

	return secretBinding.GetSecretRefName(), nil
}

func (s *ResolveSubscriptionSecretStep) provisioningAttributesFromOperationData(operation internal.Operation) *rules.ProvisioningAttributes {
	return &rules.ProvisioningAttributes{
		Plan:              broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
		PlatformRegion:    operation.ProvisioningParameters.PlatformRegion,
		HyperscalerRegion: operation.ProviderValues.Region,
		Hyperscaler:       operation.ProviderValues.ProviderType,
	}
}

func (s *ResolveSubscriptionSecretStep) matchProvisioningAttributesToRule(attr *rules.ProvisioningAttributes) (ParsedRule, error) {
	result, found := s.rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return nil, fmt.Errorf("no matching rule for provisioning attributes %q", attr)
	}
	return result, nil
}

func (s *ResolveSubscriptionSecretStep) createLabelSelectorBuilder(parsedRule ParsedRule, tenantName string) *LabelSelectorBuilder {
	b := NewLabelSelectorBuilder()
	b.With(fmt.Sprintf(hyperscalerTypeReqFmt, parsedRule.Hyperscaler()))

	if parsedRule.IsEUAccess() {
		b.With(euAccessReq)
	} else {
		b.With(notEUAccessReq)
	}

	if parsedRule.IsShared() {
		b.With(sharedReq)
		b.SaveBase()
		return b
	}

	b.With(notDirtyReq)
	b.SaveBase()

	b.With(fmt.Sprintf(tenantNameReqFmt, tenantName))

	return b
}

func (s *ResolveSubscriptionSecretStep) getSharedSecretName(labelSelector string) (string, error) {
	secretBinding, err := s.getSharedSecretBinding(labelSelector)
	if err != nil {
		return "", fmt.Errorf("while getting secret binding with selector %q: %w", labelSelector, err)
	}

	return secretBinding.GetSecretRefName(), nil
}

func (s *ResolveSubscriptionSecretStep) getSharedSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	secretBindings, err := s.gardenerClient.GetSecretBindings(labelSelector)
	if err != nil {
		return nil, err
	}
	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	secretBinding, err := s.gardenerClient.GetLeastUsedSecretBindingFromSecretBindings(secretBindings.Items)
	if err != nil {
		return nil, fmt.Errorf("while getting least used secret binding: %w", err)
	}

	return secretBinding, nil
}

func (s *ResolveSubscriptionSecretStep) getSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	secretBindings, err := s.gardenerClient.GetSecretBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting secret bindings with selector %q: %w", labelSelector, err)
	}
	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	return gardener.NewSecretBinding(secretBindings.Items[0]), nil
}

func (s *ResolveSubscriptionSecretStep) claimSecretBinding(secretBinding *gardener.SecretBinding, tenantName string) (*gardener.SecretBinding, error) {
	labels := secretBinding.GetLabels()
	labels[gardener.TenantNameLabelKey] = tenantName
	secretBinding.SetLabels(labels)

	return s.gardenerClient.UpdateSecretBinding(secretBinding)
}

func NewLabelSelectorBuilder() *LabelSelectorBuilder {
	return &LabelSelectorBuilder{}
}

func (b *LabelSelectorBuilder) ForSecretBindingClaim() {
	b.With(notSharedReq)
	b.With(notTenantNamedReq)
}

func (b *LabelSelectorBuilder) With(s string) {
	if b.Len() == 0 {
		b.WriteString(s)
		return
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
