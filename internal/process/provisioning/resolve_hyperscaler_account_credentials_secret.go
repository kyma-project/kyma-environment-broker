package provisioning

import (
	"context"
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

	notSharedSelector = gardener.SharedLabelKey + `!=true`

	notDirtySelector       = `!` + gardener.DirtyLabelKey
	notInternalSelector    = `!` + gardener.InternalLabelKey
	notEUAccessSelector    = `!` + gardener.EUAccessLabelKey
	notTenantNamedSelector = `!` + gardener.TenantNameLabelKey
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
	mux              sync.Mutex
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
	targetSecretBindingName, err := s.resolveSecretBindingName(operation, log)
	if err != nil {
		msg := fmt.Sprintf("resolving secret binding name")
		return s.operationManager.RetryOperation(operation, msg, err, 10*time.Second, time.Minute, log)
	}

	if targetSecretBindingName == "" {
		return s.operationManager.OperationFailed(operation, "failed to determine secret binding name", fmt.Errorf("target secret binding name is empty"), log)
	}

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProvisioningParameters.Parameters.TargetSecret = &targetSecretBindingName
	}, log)
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) resolveSecretBindingName(operation internal.Operation, log *slog.Logger) (string, error) {
	targetSecretBindingName := ""

	attr := s.provisioningAttributesFromOperationData(operation)
	s.rulesService.Match(attr)

	var hapParserResult HAPParserResult
	labelSelectorBuilder := s.createLabelSelectorBuilder(hapParserResult, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	secretBindings, err := s.getSecretBindings(labelSelectorBuilder.String())
	if err != nil {
		return "", fmt.Errorf("while listing secret bindings with selector %q: %w", labelSelectorBuilder.String(), err)
	}
	switch {
	case hapParserResult.IsShared():
		targetSecretBindingName, err = s.getLeastUsedSecretBindingName(secretBindings)
		if err != nil {
			return "", fmt.Errorf("unable to resolve the least used secret binding for global account ID %s on hyperscaler %s: %w",
				operation.ProvisioningParameters.ErsContext.GlobalAccountID, hapParserResult.HyperscalerType(), err)
		}
	case secretBindings != nil && len(secretBindings.Items) > 0:
		targetSecretBindingName = gardener.NewSecretBinding(secretBindings.Items[0]).GetSecretRefName()
	case secretBindings != nil && len(secretBindings.Items) == 0:
		labelSelectorBuilder.RevertToBase()
		labelSelectorBuilder.With(notSharedSelector)
		labelSelectorBuilder.With(notTenantNamedSelector)
		secretBindings, err = s.getSecretBindings(labelSelectorBuilder.String())
		if err != nil {
			return "", fmt.Errorf("while listing secret bindings with selector %q: %w", labelSelectorBuilder.String(), err)
		}
		if secretBindings == nil || len(secretBindings.Items) == 0 {
			return "", fmt.Errorf("failed to find unassigned secret binding for global account ID %s on hyperscaler %s: %w",
				operation.ProvisioningParameters.ErsContext.GlobalAccountID, hapParserResult.HyperscalerType(), err)
		}
		secretBinding := &gardener.SecretBinding{Unstructured: secretBindings.Items[0]}
		labels := secretBinding.GetLabels()
		labels["tenantName"] = operation.ProvisioningParameters.ErsContext.GlobalAccountID
		secretBinding.SetLabels(labels)
		// mutex
		updatedSecretBinding, err := s.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(s.gardenerClient.Namespace()).Update(context.Background(), &secretBinding.Unstructured, metav1.UpdateOptions{})
		if err != nil {
			return "", fmt.Errorf("while updating secret binding with tenantName: %s: %w", operation.ProvisioningParameters.ErsContext.GlobalAccountID, err)
		}
		targetSecretBindingName = gardener.NewSecretBinding(*updatedSecretBinding).GetSecretRefName()
	}
	return targetSecretBindingName, nil
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

func (s *ResolveHyperscalerAccountCredentialsSecretStep) getLeastUsedSecretBindingName(secretBindings *unstructured.UnstructuredList) (string, error) {
	secretBinding, err := s.getLeastUsedSecretBinding(secretBindings.Items)
	if err != nil {
		return "", fmt.Errorf("while getting least used secret binding: %w", err)
	}

	return secretBinding.GetSecretRefName(), nil
}

func (s *ResolveHyperscalerAccountCredentialsSecretStep) getLeastUsedSecretBinding(secretBindings []unstructured.Unstructured) (*gardener.SecretBinding, error) {
	usageCount := make(map[string]int, len(secretBindings))
	for _, s := range secretBindings {
		usageCount[s.GetName()] = 0
	}

	shoots, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(s.gardenerClient.Namespace()).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("while listing shoots: %w", err)
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return &gardener.SecretBinding{Unstructured: secretBindings[0]}, nil
	}

	for _, shoot := range shoots.Items {
		s := gardener.Shoot{Unstructured: shoot}
		count, found := usageCount[s.GetSpecSecretBindingName()]
		if !found {
			continue
		}

		usageCount[s.GetSpecSecretBindingName()] = count + 1
	}

	min := usageCount[secretBindings[0].GetName()]
	minIndex := 0

	for i, sb := range secretBindings {
		if usageCount[sb.GetName()] < min {
			min = usageCount[sb.GetName()]
			minIndex = i
		}
	}

	return &gardener.SecretBinding{Unstructured: secretBindings[minIndex]}, nil
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
