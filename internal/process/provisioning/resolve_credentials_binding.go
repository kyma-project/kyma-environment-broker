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
		// Case if there are no unassigned secrets, we want to use the error message defined in the step instead of the generic one from the error type
		if lastErr, ok := err.(kebError.LastError); ok && lastErr.Component == kebError.AccountPoolDependency {
			msg = lastErr.Message
		}
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
		return s.getSharedCredentialsName(selectorForExistingSubscription, log)
	}

	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID
	if s.multiAccountConfig.IsGlobalAccountAllowed(globalAccountID) {
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

func (s *ResolveCredentialsBindingStep) getSharedCredentialsName(labelSelector string, log *slog.Logger) (string, error) {
	credentialsBinding, err := s.getSharedCredentialsBinding(labelSelector)
	if err != nil {
		if kebError.IsNotFoundError(err) {
			log.Error(fmt.Sprintf("failed to find unassigned credentials binding with selector %q", labelSelector))
			return "", kebError.LastError{
				Message:   "Currently, no unassigned provider accounts are available. Please contact us for further assistance.",
				Reason:    kebError.KEBInternalCode,
				Component: kebError.AccountPoolDependency,
			}
		}
		return "", fmt.Errorf("while getting credentials binding with selector %q: %w", labelSelector, err)
	}

	return credentialsBinding.GetName(), nil
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
		return nil, fmt.Errorf("while getting least used credentials binding: %w", err)
	}

	return credentialsBinding, nil
}

func (s *ResolveCredentialsBindingStep) getCredentialsBinding(labelSelector string) (*gardener.CredentialsBinding, error) {
	credentialsBindings, err := s.gardenerClient.GetCredentialsBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting credentials bindings with selector %q: %w", labelSelector, err)
	}
	if credentialsBindings == nil || len(credentialsBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	return gardener.NewCredentialsBinding(credentialsBindings.Items[0]), nil
}

func (s *ResolveCredentialsBindingStep) claimCredentialsBinding(credentialsBinding *gardener.CredentialsBinding, tenantName string) (*gardener.CredentialsBinding, error) {
	labels := credentialsBinding.GetLabels()
	labels[gardener.TenantNameLabelKey] = tenantName
	credentialsBinding.SetLabels(labels)

	return s.gardenerClient.UpdateCredentialsBinding(credentialsBinding)
}

func (s *ResolveCredentialsBindingStep) updateInstance(id, subscriptionSecretName string) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return err
	}
	instance.SubscriptionSecretName = subscriptionSecretName
	_, err = s.instanceStorage.Update(*instance)
	return err
}

func (s *ResolveCredentialsBindingStep) resolveWithMultiAccountSupport(operation internal.Operation, selectorForExistingSubscription string, labelSelectorBuilder *subscriptions.LabelSelectorBuilder, log *slog.Logger) (string, error) {
	globalAccountID := operation.ProvisioningParameters.ErsContext.GlobalAccountID

	allBindings, err := s.gardenerClient.GetCredentialsBindings(selectorForExistingSubscription)
	if err != nil {
		return "", fmt.Errorf("while getting credentials bindings for tenant %s: %w", globalAccountID, err)
	}
	hyperscalerAccountLimit := s.multiAccountConfig.LimitForProvider(operation.ProviderValues.ProviderType)

	if allBindings != nil && len(allBindings.Items) > 0 {
		bindingNames := make([]string, len(allBindings.Items))
		for i, binding := range allBindings.Items {
			bindingNames[i] = binding.GetName()
		}
		log.Info(fmt.Sprintf("found %d credentials binding(s) for GA %s, provider limit %d, bindings: %v", len(allBindings.Items), globalAccountID, hyperscalerAccountLimit, bindingNames))

		instancesPerBinding, err := s.instanceStorage.GetInstanceCountPerBinding(globalAccountID, bindingNames)
		if err != nil {
			return "", fmt.Errorf("while getting instance counts per binding: %w", err)
		}

		if selectedBinding, count := s.selectBindingBelowLimit(bindingNames, instancesPerBinding, hyperscalerAccountLimit, log); selectedBinding != "" {
			log.Info(fmt.Sprintf("selected credentials binding %s with %d instances (below limit %d)", selectedBinding, count, hyperscalerAccountLimit))
			return selectedBinding, nil
		}

		log.Info(fmt.Sprintf("all %d credentials bindings for GA %s are at or above limit %d, will claim new one", len(allBindings.Items), globalAccountID, hyperscalerAccountLimit))
	}

	return s.claimNewCredentialsBinding(globalAccountID, labelSelectorBuilder, log)
}

// selectBindingBelowLimit finds the most populated binding that is still below the limit.
// Bindings with tenantName label but 0 instances in KEB DB are also considered
func (s *ResolveCredentialsBindingStep) selectBindingBelowLimit(bindingNames []string, instancesPerBinding map[string]int, limit int, log *slog.Logger) (string, int) {
	selected := ""
	selectedCount := -1
	for _, name := range bindingNames {
		count := instancesPerBinding[name]
		log.Info(fmt.Sprintf("credentials binding %s has %d instances", name, count))
		if count < limit && count > selectedCount {
			selected = name
			selectedCount = count
		}
	}
	return selected, selectedCount
}

func (s *ResolveCredentialsBindingStep) claimNewCredentialsBinding(globalAccountID string, labelSelectorBuilder *subscriptions.LabelSelectorBuilder, log *slog.Logger) (string, error) {
	log.Info(fmt.Sprintf("no credentials binding found for tenant: %q", globalAccountID))

	s.mu.Lock()
	defer s.mu.Unlock()

	selectorForSBClaim := labelSelectorBuilder.BuildForSecretBindingClaim()

	log.Info(fmt.Sprintf("getting credentials binding with selector %q", selectorForSBClaim))
	credentialsBinding, err := s.getCredentialsBinding(selectorForSBClaim)
	if err != nil {
		if kebError.IsNotFoundError(err) {
			log.Error(fmt.Sprintf("failed to find unassigned credentials binding with selector %q", selectorForSBClaim))
			return "", kebError.LastError{
				Message:   "Currently, no unassigned provider accounts are available. Please contact us for further assistance.",
				Reason:    kebError.KEBInternalCode,
				Component: kebError.AccountPoolDependency,
			}
		}
		return "", err
	}

	log.Info(fmt.Sprintf("claiming credentials binding %s for tenant %q", credentialsBinding.GetName(), globalAccountID))
	credentialsBinding, err = s.claimCredentialsBinding(credentialsBinding, globalAccountID)
	if err != nil {
		return "", fmt.Errorf("while claiming credentials binding for tenant: %s: %w", globalAccountID, err)
	}

	return credentialsBinding.GetName(), nil
}
