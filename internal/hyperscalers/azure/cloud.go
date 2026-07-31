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

// probeFunc is a function that tests whether credentials authenticate against a given cloud.
// Replaceable in tests to avoid real Azure API calls.
type probeFunc func(ctx context.Context, creds AzureCredentials, cfg cloud.Configuration) bool

var probeOrder = []struct {
	name string
	cfg  cloud.Configuration
}{
	{"public", cloud.AzurePublic},
	{"china", cloud.AzureChina},
	{"usgov", cloud.AzureGovernment},
}

// ResolveCloudConfig returns the cloud.Configuration to use for the given credentials.
// When configName is non-empty it maps directly to the SDK constant — no network calls.
// When configName is empty it probes Public → China → US Gov and returns the first that succeeds.
// Intended to be called once at KEB startup before the cache and HTTP server are started.
func ResolveCloudConfig(ctx context.Context, creds AzureCredentials, configName string) (cloud.Configuration, error) {
	return resolveCloudConfig(ctx, creds, configName, probeCloud)
}

func resolveCloudConfig(ctx context.Context, creds AzureCredentials, configName string, probe probeFunc) (cloud.Configuration, error) {
	if configName != "" {
		cfg, err := CloudConfigFromName(configName)
		if err != nil {
			return cloud.Configuration{}, err
		}
		slog.Info("Azure cloud configured explicitly", "cloud", configName)
		return cfg, nil
	}

	for _, p := range probeOrder {
		if probe(ctx, creds, p.cfg) {
			slog.Info("Azure cloud auto-discovered", "cloud", p.name)
			return p.cfg, nil
		}
		slog.Info("Azure cloud probe failed", "cloud", p.name)
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
	svc, ok := cfg.Services[cloud.ResourceManager]
	if !ok || svc.Audience == "" {
		return false
	}
	_, err = credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{svc.Audience + "/.default"},
	})
	return err == nil
}
