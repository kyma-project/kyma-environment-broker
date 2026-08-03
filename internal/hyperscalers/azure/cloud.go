package azure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
)

const azureCloudProbeTimeout = 10 * time.Second

// CloudConfigFromName maps a human-readable cloud name to the corresponding SDK constant.
// Accepted values: hyperscaler.AzureCloudPublic, hyperscaler.AzureCloudChina, hyperscaler.AzureCloudUSGovernment.
func CloudConfigFromName(name string) (cloud.Configuration, error) {
	switch name {
	case hyperscaler.AzureCloudPublic:
		return cloud.AzurePublic, nil
	case hyperscaler.AzureCloudChina:
		return cloud.AzureChina, nil
	case hyperscaler.AzureCloudUSGovernment:
		return cloud.AzureGovernment, nil
	default:
		return cloud.Configuration{}, fmt.Errorf("unknown Azure cloud %q: must be one of %s, %s, %s", name, hyperscaler.AzureCloudPublic, hyperscaler.AzureCloudChina, hyperscaler.AzureCloudUSGovernment)
	}
}

// probeFunc is a function that tests whether credentials authenticate against a given cloud.
// Replaceable in tests to avoid real Azure API calls.
type probeFunc func(ctx context.Context, creds AzureCredentials, cfg cloud.Configuration) bool

var probeOrder = []struct {
	name string
	cfg  cloud.Configuration
}{
	{hyperscaler.AzureCloudPublic, cloud.AzurePublic},
	{hyperscaler.AzureCloudChina, cloud.AzureChina},
	{hyperscaler.AzureCloudUSGovernment, cloud.AzureGovernment},
}

// ResolveCloudConfig probes Public → China → US Gov and returns the first cloud
// that accepts the credentials. Intended to be called once at KEB startup.
func ResolveCloudConfig(ctx context.Context, creds AzureCredentials) (cloud.Configuration, error) {
	return resolveCloudConfig(ctx, creds, probeCloud)
}

func resolveCloudConfig(ctx context.Context, creds AzureCredentials, probe probeFunc) (cloud.Configuration, error) {
	for _, p := range probeOrder {
		if probe(ctx, creds, p.cfg) {
			slog.Info("Azure cloud auto-discovered", "cloud", p.name)
			return p.cfg, nil
		}
		slog.Info("Azure cloud probe failed", "cloud", p.name)
	}
	return cloud.Configuration{}, fmt.Errorf("Azure cloud auto-discovery failed: credentials did not authenticate against any cloud (%s, %s, %s)", hyperscaler.AzureCloudPublic, hyperscaler.AzureCloudChina, hyperscaler.AzureCloudUSGovernment)
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
		slog.Info("Azure cloud probe: failed to create credential", "error", err)
		return false
	}
	svc, ok := cfg.Services[cloud.ResourceManager]
	if !ok || svc.Audience == "" {
		slog.Info("Azure cloud probe: ResourceManager service not found in cloud configuration")
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, azureCloudProbeTimeout)
	defer cancel()
	_, err = credential.GetToken(probeCtx, policy.TokenRequestOptions{
		Scopes: []string{svc.Audience + "/.default"},
	})
	if err != nil {
		slog.Info("Azure cloud probe: token request failed", "error", err)
		return false
	}
	return true
}
