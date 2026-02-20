package provisioning

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/multiaccount"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	runtimepkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type ResolveCredentialsBindingStep struct {
	operationManager   *process.OperationManager
	gardenerClient     *gardener.Client
	instanceStorage    storage.Instances
	rulesService       *rules.RulesService
	stepRetryTuple     internal.RetryTuple
	mu                 sync.Mutex
	multiAccountConfig *multiaccount.MultiAccountConfig
}

func NewResolveCredentialsBindingStep(brokerStorage storage.BrokerStorage, gardenerClient *gardener.Client, rulesService *rules.RulesService, stepRetryTuple internal.RetryTuple, multiAccountConfig *multiaccount.MultiAccountConfig) *ResolveCredentialsBindingStep {
	step := &ResolveCredentialsBindingStep{
		instanceStorage:    brokerStorage.Instances(),
		gardenerClient:     gardenerClient,
		rulesService:       rulesService,
		stepRetryTuple:     stepRetryTuple,
		multiAccountConfig: multiAccountConfig,
	}
	step.operationManager = process.NewOperationManager(brokerStorage.Operations(), step.Name(), kebError.AccountPoolDependency)
	return step
}

func (s *ResolveCredentialsBindingStep) Name() string {
	return "Resolve_Credentials_Binding"
}

func (s *ResolveCredentialsBindingStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.ProvisioningParameters.Parameters.TargetSecret != nil && *operation.ProvisioningParameters.Parameters.TargetSecret != "" {
		log.Info("target secret is already set, skipping resolve step")
		return operation, 0, nil
	}
	targetSecretName, err := s.resolveSecretName(operation, log)
	if err != nil {
		msg := "resolving secret name"
		return s.operationManager.RetryOperation(operation, msg, err, s.stepRetryTuple.Interval, s.stepRetryTuple.Timeout, log)
	}

	if targetSecretName == "" {
		return s.operationManager.OperationFailed(operation, "failed to determine secret name", fmt.Errorf("target secret name is empty"), log)
	}
	log.Info(fmt.Sprintf("resolved credentials binding name: %s", targetSecretName))

	err = s.updateInstance(operation.InstanceID, targetSecretName)
	if err != nil {
		log.Error(fmt.Sprintf("failed to update instance with subscription secret name: %s", err.Error()))
		return s.operationManager.RetryOperation(operation, "updating instance", err, s.stepRetryTuple.Interval, s.stepRetryTuple.Timeout, log)
	}

	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProvisioningParameters.Parameters.TargetSecret = &targetSecretName
	}, log)
}

func (s *ResolveCredentialsBindingStep) resolveSecretName(operation internal.Operation, log *slog.Logger) (string, error) {
	attr := s.provisioningAttributesFromOperationData(operation)

	log.Info(fmt.Sprintf("matching provisioning attributes %q to filtering rule", attr))
	parsedRule, err := s.matchProvisioningAttributesToRule(attr)
	if err != nil {
		return "", err
	}

	log.Info(fmt.Sprintf("matched rule: %q", parsedRule.Rule()))

	labelSelectorBuilder := subscriptions.NewLabelSelectorFromRuleset(parsedRule)
	selectorForExistingSubscription := labelSelectorBuilder.BuildForTenantMatching(operation.ProvisioningParameters.ErsContext.GlobalAccountID)

	log.Info(fmt.Sprintf("getting credentials binding with selector %q", selectorForExistingSubscription))
	if parsedRule.IsShared() {
		return s.getSharedSecretName(selectorForExistingSubscription)
	}

	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	if isGlobalAccountAllowed(s.multiAccountConfig, globalAccountID) {
		log.Info(fmt.Sprintf("multi-account support enabled for GA: %s", globalAccountID))
		return s.resolveWithMultiAccountSupport(operation, selectorForExistingSubscription, labelSelectorBuilder, log)
	}

	credentialsBinding, err := s.getCredentialsBinding(selectorForExistingSubscription)
	if err != nil && !kebError.IsNotFoundError(err) {
		return "", err
	}

	if credentialsBinding != nil {
		return credentialsBinding.GetName(), nil
	}

	return s.claimNewCredentialsBinding(operation.ProvisioningParameters.ErsContext.GlobalAccountID, labelSelectorBuilder, log)
}

func (s *ResolveCredentialsBindingStep) provisioningAttributesFromOperationData(operation internal.Operation) *rules.ProvisioningAttributes {
	return &rules.ProvisioningAttributes{
		Plan:              broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(operation.ProvisioningParameters.PlanID)),
		PlatformRegion:    operation.ProvisioningParameters.PlatformRegion,
		HyperscalerRegion: operation.ProviderValues.Region,
		Hyperscaler:       operation.ProviderValues.ProviderType,
	}
}

func (s *ResolveCredentialsBindingStep) matchProvisioningAttributesToRule(attr *rules.ProvisioningAttributes) (subscriptions.ParsedRule, error) {
	result, found := s.rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return nil, fmt.Errorf("no matching rule for provisioning attributes %q", attr)
	}
	return result, nil
}

func (s *ResolveCredentialsBindingStep) getSharedSecretName(labelSelector string) (string, error) {
	secretBinding, err := s.getSharedCredentialsBinding(labelSelector)
	if err != nil {
		return "", fmt.Errorf("while getting secret binding with selector %q: %w", labelSelector, err)
	}

	return secretBinding.GetName(), nil
}

func (s *ResolveCredentialsBindingStep) getSharedCredentialsBinding(labelSelector string) (*gardener.CredentialsBinding, error) {
	credentialsBindings, err := s.gardenerClient.GetCredentialsBindings(labelSelector)
	if err != nil {
		return nil, err
	}
	if credentialsBindings == nil || len(credentialsBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	credentialsBinding, err := s.gardenerClient.GetLeastUsedCredentialsBindingFromSecretBindings(credentialsBindings.Items)
	if err != nil {
		return nil, fmt.Errorf("while getting least used secret binding: %w", err)
	}

	return credentialsBinding, nil
}

func (s *ResolveCredentialsBindingStep) getCredentialsBinding(labelSelector string) (*gardener.CredentialsBinding, error) {
	secretBindings, err := s.gardenerClient.GetCredentialsBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting secret bindings with selector %q: %w", labelSelector, err)
	}
	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	return gardener.NewCredentialsBinding(secretBindings.Items[0]), nil
}

func (s *ResolveCredentialsBindingStep) claimCredentialsBinding(credentialsBinding *gardener.CredentialsBinding, tenantName string) (*gardener.CredentialsBinding, error) {
	labels := credentialsBinding.GetLabels()
	labels[gardener.TenantNameLabelKey] = tenantName
	credentialsBinding.SetLabels(labels)

	return s.gardenerClient.UpdateCredentialsBinding(credentialsBinding)
}

func (step *ResolveCredentialsBindingStep) updateInstance(id, subscriptionSecretName string) error {
	instance, err := step.instanceStorage.GetByID(id)
	if err != nil {
		return err
	}
	instance.SubscriptionSecretName = subscriptionSecretName
	_, err = step.instanceStorage.Update(*instance)
	return err
}

func (s *ResolveCredentialsBindingStep) resolveWithMultiAccountSupport(operation internal.Operation, selectorForExistingSubscription string, labelSelectorBuilder *subscriptions.LabelSelectorBuilder, log *slog.Logger) (string, error) {
	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID

	allBindings, err := s.gardenerClient.GetCredentialsBindings(selectorForExistingSubscription)
	if err != nil {
		return "", fmt.Errorf("while getting credentials bindings for tenant %s: %w", globalAccountID, err)
	}
	hyperscalerAccountLimit := getLimitForProvider(s.multiAccountConfig, operation.ProviderValues.ProviderType)
	log.Info(fmt.Sprintf("found %d credentials binding(s) for GA %s, provider limit: %d", len(allBindings.Items), globalAccountID, hyperscalerAccountLimit))

	if allBindings != nil && len(allBindings.Items) > 0 {
		credentialsBinding, err := s.gardenerClient.GetMostPopulatedCredentialsBindingBelowLimit(allBindings.Items, hyperscalerAccountLimit)
		if err != nil {
			return "", fmt.Errorf("while selecting credentials binding: %w", err)
		}

		if credentialsBinding != nil {
			log.Info(fmt.Sprintf("selected credentials binding %s (below limit %d)", credentialsBinding.GetName(), hyperscalerAccountLimit))
			return credentialsBinding.GetName(), nil
		}

		log.Info(fmt.Sprintf("all %d credentials bindings for GA %s are at or above limit %d, will claim new one", len(allBindings.Items), globalAccountID, hyperscalerAccountLimit))
	}

	return s.claimNewCredentialsBinding(globalAccountID, labelSelectorBuilder, log)
}

func (s *ResolveCredentialsBindingStep) claimNewCredentialsBinding(globalAccountID string, labelSelectorBuilder *subscriptions.LabelSelectorBuilder, log *slog.Logger) (string, error) {
	log.Info(fmt.Sprintf("no credentials binding found for tenant: %q", globalAccountID))

	s.mu.Lock()
	defer s.mu.Unlock()

	selectorForSBClaim := labelSelectorBuilder.BuildForSecretBindingClaim()

	log.Info(fmt.Sprintf("getting secret binding with selector %q", selectorForSBClaim))
	credentialsBinding, err := s.getCredentialsBinding(selectorForSBClaim)
	if err != nil {
		if kebError.IsNotFoundError(err) {
			return "", fmt.Errorf("failed to find unassigned secret binding with selector %q", selectorForSBClaim)
		}
		return "", err
	}

	log.Info(fmt.Sprintf("claiming credentials binding for tenant %q", globalAccountID))
	credentialsBinding, err = s.claimCredentialsBinding(credentialsBinding, globalAccountID)
	if err != nil {
		return "", fmt.Errorf("while claiming secret binding for tenant: %s: %w", globalAccountID, err)
	}

	return credentialsBinding.GetName(), nil
}

func isMultiAccountEnabled(config *multiaccount.MultiAccountConfig) bool {
	return config != nil && len(config.AllowedGlobalAccounts) > 0
}

func isGlobalAccountAllowed(config *multiaccount.MultiAccountConfig, globalAccountID string) bool {
	if !isMultiAccountEnabled(config) {
		return false
	}

	for _, ga := range config.AllowedGlobalAccounts {
		if ga == "*" || ga == globalAccountID {
			return true
		}
	}

	return false
}

func getLimitForProvider(config *multiaccount.MultiAccountConfig, providerType string) int {
	if config == nil {
		return 0
	}
	cp := runtimepkg.CloudProviderFromString(providerType)

	var limit int
	switch cp {
	case runtimepkg.AWS:
		limit = config.Limits.AWS
	case runtimepkg.GCP:
		limit = config.Limits.GCP
	case runtimepkg.Azure:
		limit = config.Limits.Azure
	case runtimepkg.SapConvergedCloud:
		limit = config.Limits.OpenStack
	case runtimepkg.Alicloud:
		limit = config.Limits.AliCloud
	default:
		limit = 0
	}

	if limit == 0 {
		return config.Limits.Default
	}

	return limit
}
