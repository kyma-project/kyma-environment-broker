package azure

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// CloudConfigFromName maps a human-readable cloud name to the corresponding SDK constant.
// Accepted values: "public", "china", "usgov".
func CloudConfigFromName(name string) (cloud.Configuration, error) {
	switch name {
	case "public":
		return cloud.AzurePublic, nil
	case "china":
		return cloud.AzureChina, nil
	case "usgov":
		return cloud.AzureGovernment, nil
	default:
		return cloud.Configuration{}, fmt.Errorf("unknown Azure cloud %q: must be one of public, china, usgov", name)
	}
}

// ResolveCloudConfig returns the cloud.Configuration to use for the given credentials.
// When configName is non-empty it maps directly to the SDK constant — no network calls.
// When configName is empty it probes Public → China → US Gov and returns the first that succeeds.
// Intended to be called once at KEB startup before the cache and HTTP server are started.
func ResolveCloudConfig(ctx context.Context, creds AzureCredentials, configName string) (cloud.Configuration, error) {
	if configName != "" {
		cfg, err := CloudConfigFromName(configName)
		if err != nil {
			return cloud.Configuration{}, err
		}
		slog.Info("Azure cloud configured explicitly", "cloud", configName)
		return cfg, nil
	}

	var probeOrder = []cloud.Configuration{
		cloud.AzurePublic,
		cloud.AzureChina,
		cloud.AzureGovernment,
	}
	var probeOrderNames = []string{"public", "china", "usgov"}

	for i, cfg := range probeOrder {
		if probeCloud(ctx, creds, cfg) {
			slog.Info("Azure cloud auto-discovered", "cloud", probeOrderNames[i])
			return cfg, nil
		}
		slog.Info("Azure cloud probe failed", "cloud", probeOrderNames[i])
	}
	return cloud.Configuration{}, fmt.Errorf("Azure cloud auto-discovery failed: credentials did not authenticate against any cloud (public, china, usgov)")
}

// probeCloud attempts to acquire an Azure ARM token using the given cloud configuration.
// Returns true if authentication succeeds.
func probeCloud(ctx context.Context, creds AzureCredentials, cfg cloud.Configuration) bool {
	credential, err := azidentity.NewClientSecretCredential(
		creds.TenantID, creds.ClientID, creds.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{
			ClientOptions: azcore.ClientOptions{Cloud: cfg},
		},
	)
	if err != nil {
		return false
	}
	_, err = credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{cfg.Services[cloud.ResourceManager].Audience + "/.default"},
	})
	return err == nil
}
