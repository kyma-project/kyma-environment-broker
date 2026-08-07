package main

import (
	"context"
	"fmt"
	"log/slog"

	azurecloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers"
	azurehyperscaler "github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/azure"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// buildAzureSecretFetcher returns a SecretFetcher that fetches a valid Azure
// credentials secret from Gardener on each call, retrying up to 4 different
// credential bindings if one is missing or malformed. This ensures credential
// rotation is picked up on every cache refresh without restarting KEB.
func buildAzureSecretFetcher(gardenerClient *gardener.Client, rulesService *rules.RulesService, cloudConfig azurecloud.Configuration, log *slog.Logger) (azurehyperscaler.SecretFetcher, error) {
	attr := &rules.ProvisioningAttributes{
		Plan:        "azure",
		Hyperscaler: "azure",
	}
	matchedRule, found := rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return nil, fmt.Errorf("no matching rule for azure hyperscaler")
	}
	labelSelector := subscriptions.NewLabelSelectorFromRuleset(matchedRule).BuildAnySubscription()

	return func() (azurehyperscaler.AzureCredentials, error) {
		return fetchAzureCredentials(gardenerClient, labelSelector, cloudConfig, log)
	}, nil
}

func fetchAzureCredentials(gardenerClient *gardener.Client, labelSelector string, cloudConfig azurecloud.Configuration, log *slog.Logger) (azurehyperscaler.AzureCredentials, error) {
	return hyperscalers.WithRetry(context.Background(), gardenerClient, labelSelector, log,
		func(ctx context.Context, cb *gardener.CredentialsBinding, secret *unstructured.Unstructured) (azurehyperscaler.AzureCredentials, error) {
			log.Info("fetching Azure credentials", "credentialBinding", cb.GetName())
			creds, err := azurehyperscaler.ExtractCredentials(secret)
			if err != nil {
				return azurehyperscaler.AzureCredentials{}, fmt.Errorf("failed to extract Azure credentials from binding %s: %w", cb.GetName(), err)
			}
			if !azurehyperscaler.ProbeCloud(ctx, creds, cloudConfig) {
				return azurehyperscaler.AzureCredentials{}, fmt.Errorf("credentials from binding %s failed cloud probe", cb.GetName())
			}
			return creds, nil
		})
}

// resolveAzureCloudConfig determines the Azure cloud environment for zone discovery API calls.
// When clientConfiguration is set explicitly, it is used directly with no network calls.
// Otherwise the cloud is auto-discovered by probing Public → China → US Gov at startup,
// retrying with up to 4 different credential bindings if a binding's credentials are invalid.
func resolveAzureCloudConfig(ctx context.Context, providerSpec *configuration.ProviderSpec, gardenerClient *gardener.Client, rulesService *rules.RulesService, log *slog.Logger) (azurecloud.Configuration, error) {
	if configName := providerSpec.AzureClientConfiguration(); configName != "" {
		cfg, err := azurehyperscaler.CloudConfigFromName(configName)
		if err != nil {
			return azurecloud.Configuration{}, err
		}
		log.Info("Azure cloud configured explicitly", "cloud", configName)
		return cfg, nil
	}

	attr := &rules.ProvisioningAttributes{
		Plan:        "azure",
		Hyperscaler: "azure",
	}
	matchedRule, found := rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return azurecloud.Configuration{}, fmt.Errorf("Azure cloud auto-discovery not possible: no matching rule for azure hyperscaler")
	}
	labelSelector := subscriptions.NewLabelSelectorFromRuleset(matchedRule).BuildAnySubscription()

	return hyperscalers.WithRetry(ctx, gardenerClient, labelSelector, log,
		func(ctx context.Context, cb *gardener.CredentialsBinding, secret *unstructured.Unstructured) (azurecloud.Configuration, error) {
			creds, err := azurehyperscaler.ExtractCredentials(secret)
			if err != nil {
				return azurecloud.Configuration{}, fmt.Errorf("failed to extract Azure credentials from binding %s: %w", cb.GetName(), err)
			}
			return azurehyperscaler.ResolveCloudConfig(ctx, creds)
		})
}
