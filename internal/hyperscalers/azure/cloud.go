package azure

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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
//
// When configName is non-empty it maps directly to the SDK constant via CloudConfigFromName
// with no network calls.
//
// When configName is empty, it auto-discovers the cloud by probing Public → China → US Gov
// using a single OAuth token request per candidate. The result is cached for the lifetime of
// the process (sync.Once), because the Azure cloud environment never changes for a given
// deployment. Concurrent callers block until the first probe completes.
func ResolveCloudConfig(ctx context.Context, creds AzureCredentials, configName string) (cloud.Configuration, error) {
	if configName != "" {
		return CloudConfigFromName(configName)
	}
	return discoverCloudConfig(ctx, creds)
}

// probeOrder defines the discovery order: public first (most common), then china, then usgov.
var probeOrder = []cloud.Configuration{
	cloud.AzurePublic,
	cloud.AzureChina,
	cloud.AzureGovernment,
}

var probeOrderNames = []string{"public", "china", "usgov"}

var (
	discoverOnce      sync.Once
	discoveredCloud   cloud.Configuration
	discoverErr       error
)

func discoverCloudConfig(ctx context.Context, creds AzureCredentials) (cloud.Configuration, error) {
	discoverOnce.Do(func() {
		for i, cfg := range probeOrder {
			if probeCloud(ctx, creds, cfg) {
				slog.Info("Azure cloud auto-discovered", "cloud", probeOrderNames[i])
				discoveredCloud = cfg
				return
			}
			slog.Info("Azure cloud probe failed", "cloud", probeOrderNames[i])
		}
		discoverErr = fmt.Errorf("Azure cloud auto-discovery failed: credentials did not authenticate against any cloud (public, china, usgov)")
	})
	return discoveredCloud, discoverErr
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
	// Request a token for the ARM audience of the candidate cloud.
	// This is a cheap single HTTPS call to the cloud's token endpoint.
	_, err = credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{cfg.Services[cloud.ResourceManager].Audience + "/.default"},
	})
	return err == nil
}
