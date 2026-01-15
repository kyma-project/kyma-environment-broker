package provisioning

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type ResolveSubscriptionSecretStep struct {
	operationManager *process.OperationManager
	gardenerClient   *gardener.Client
	opStorage        storage.Operations
	instanceStorage  storage.Instances
	rulesService     *rules.RulesService
	stepRetryTuple   internal.RetryTuple
	mu               sync.Mutex
}

func NewResolveSubscriptionSecretStep(brokerStorage storage.BrokerStorage, gardenerClient *gardener.Client, rulesService *rules.RulesService, stepRetryTuple internal.RetryTuple) *ResolveSubscriptionSecretStep {
	step := &ResolveSubscriptionSecretStep{
		opStorage:       brokerStorage.Operations(),
		instanceStorage: brokerStorage.Instances(),
		gardenerClient:  gardenerClient,
		rulesService:    rulesService,
		stepRetryTuple:  stepRetryTuple,
	}
	step.operationManager = process.NewOperationManager(brokerStorage.Operations(), step.Name(), kebError.AccountPoolDependency)
	return step
}

func (step *ResolveSubscriptionSecretStep) Name() string {
	return "Resolve_Subscription_Secret"
}

func (step *ResolveSubscriptionSecretStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.ProvisioningParameters.Parameters.TargetSecret != nil && *operation.ProvisioningParameters.Parameters.TargetSecret != "" {
		log.Info("target secret is already set, skipping resolve step")
		return operation, 0, nil
	}
	targetSecretName, err := step.resolveSecretName(operation, log)
	if err != nil {
		msg := fmt.Sprintf("resolving secret name")
		return step.operationManager.RetryOperation(operation, msg, err, step.stepRetryTuple.Interval, step.stepRetryTuple.Timeout, log)
	}

	if targetSecretName == "" {
		return step.operationManager.OperationFailed(operation, "failed to determine secret name", fmt.Errorf("target secret name is empty"), log)
	}
	log.Info(fmt.Sprintf("resolved secret binding name: %step", targetSecretName))

	err = step.updateInstance(operation.InstanceID, targetSecretName)
	if err != nil {
		log.Error(fmt.Sprintf("failed to update instance with subscription secret name: %step", err.Error()))
		return step.operationManager.RetryOperation(operation, "updating instance", err, step.stepRetryTuple.Interval, step.stepRetryTuple.Timeout, log)
	}

	return step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.ProvisioningParameters.Parameters.TargetSecret = &targetSecretName
	}, log)
}

func (step *ResolveSubscriptionSecretStep) resolveSecretName(operation internal.Operation, log *slog.Logger) (string, error) {
	attr := step.provisioningAttributesFromOperationData(operation)

	log.Info(fmt.Sprintf("matching provisioning attributes %q to filtering rule", attr))
	parsedRule, err := step.matchProvisioningAttributesToRule(attr)
	if err != nil {
		return "", err
	}

	log.Info(fmt.Sprintf("matched rule: %q", parsedRule.Rule()))

	labelSelectorBuilder := subscriptions.NewLabelSelectorFromRuleset(parsedRule)
	selectorForExistingSubscription := labelSelectorBuilder.BuildForTenantMatching(operation.ProvisioningParameters.ErsContext.GlobalAccountID)

	log.Info(fmt.Sprintf("getting secret binding with selector %q", selectorForExistingSubscription))
	if parsedRule.IsShared() {
		return step.getSharedSecretName(selectorForExistingSubscription)
	}

	secretBinding, err := step.getSecretBinding(selectorForExistingSubscription)
	if err != nil && !kebError.IsNotFoundError(err) {
		return "", err
	}

	if secretBinding != nil {
		return secretBinding.GetName(), nil
	}

	log.Info(fmt.Sprintf("no secret binding found for tenant: %q", operation.ProvisioningParameters.ErsContext.GlobalAccountID))

	step.mu.Lock()
	defer step.mu.Unlock()

	selectorForSBClaim := labelSelectorBuilder.BuildForSecretBindingClaim()

	log.Info(fmt.Sprintf("getting secret binding with selector %q", selectorForSBClaim))
	secretBinding, err = step.getSecretBinding(selectorForSBClaim)
	if err != nil {
		if kebError.IsNotFoundError(err) {
			return "", fmt.Errorf("failed to find unassigned secret binding with selector %q", selectorForSBClaim)
		}
		return "", err
	}

	log.Info(fmt.Sprintf("claiming secret binding for tenant %q", operation.ProvisioningParameters.ErsContext.GlobalAccountID))
	secretBinding, err = step.claimSecretBinding(secretBinding, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	if err != nil {
		return "", fmt.Errorf("while claiming secret binding for tenant: %step: %w", operation.ProvisioningParameters.ErsContext.GlobalAccountID, err)
	}

	return secretBinding.GetName(), nil
}

func (step *ResolveSubscriptionSecretStep) provisioningAttributesFromOperationData(operation internal.Operation) *rules.ProvisioningAttributes {
	return &rules.ProvisioningAttributes{
		Plan:              broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(operation.ProvisioningParameters.PlanID)),
		PlatformRegion:    operation.ProvisioningParameters.PlatformRegion,
		HyperscalerRegion: operation.ProviderValues.Region,
		Hyperscaler:       operation.ProviderValues.ProviderType,
	}
}

func (step *ResolveSubscriptionSecretStep) matchProvisioningAttributesToRule(attr *rules.ProvisioningAttributes) (subscriptions.ParsedRule, error) {
	result, found := step.rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return nil, fmt.Errorf("no matching rule for provisioning attributes %q", attr)
	}
	return result, nil
}

func (step *ResolveSubscriptionSecretStep) getSharedSecretName(labelSelector string) (string, error) {
	secretBinding, err := step.getSharedSecretBinding(labelSelector)
	if err != nil {
		return "", fmt.Errorf("while getting secret binding with selector %q: %w", labelSelector, err)
	}

	return secretBinding.GetName(), nil
}

func (step *ResolveSubscriptionSecretStep) getSharedSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	secretBindings, err := step.gardenerClient.GetSecretBindings(labelSelector)
	if err != nil {
		return nil, err
	}
	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	secretBinding, err := step.gardenerClient.GetLeastUsedSecretBindingFromSecretBindings(secretBindings.Items)
	if err != nil {
		return nil, fmt.Errorf("while getting least used secret binding: %w", err)
	}

	return secretBinding, nil
}

func (step *ResolveSubscriptionSecretStep) getSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	secretBindings, err := step.gardenerClient.GetSecretBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting secret bindings with selector %q: %w", labelSelector, err)
	}
	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, kebError.NewNotFoundError(kebError.K8SNoMatchCode, kebError.AccountPoolDependency)
	}
	return gardener.NewSecretBinding(secretBindings.Items[0]), nil
}

func (step *ResolveSubscriptionSecretStep) claimSecretBinding(secretBinding *gardener.SecretBinding, tenantName string) (*gardener.SecretBinding, error) {
	labels := secretBinding.GetLabels()
	labels[gardener.TenantNameLabelKey] = tenantName
	secretBinding.SetLabels(labels)

	return step.gardenerClient.UpdateSecretBinding(secretBinding)
}

func (step *ResolveSubscriptionSecretStep) updateInstance(id, subscriptionSecretName string) error {
	instance, err := step.instanceStorage.GetByID(id)
	if err != nil {
		return err
	}
	instance.SubscriptionSecretName = subscriptionSecretName
	_, err = step.instanceStorage.Update(*instance)
	return err
}
