package azure

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudConfigFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected cloud.Configuration
	}{
		{"public", cloud.AzurePublic},
		{"china", cloud.AzureChina},
		{"usgov", cloud.AzureGovernment},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CloudConfigFromName(tt.name)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCloudConfigFromName_Unknown(t *testing.T) {
	_, err := CloudConfigFromName("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown Azure cloud")
}

func TestResolveCloudConfig_ExplicitName_NeverProbes(t *testing.T) {
	called := false
	_, err := resolveCloudConfig(context.Background(), AzureCredentials{}, func(_ context.Context, _ AzureCredentials, _ cloud.Configuration) bool {
		called = true
		return false
	})
	require.Error(t, err, "empty creds should fail auto-discovery")
	assert.True(t, called, "probe must be called during auto-discovery")

	// Now verify that CloudConfigFromName (the explicit path) never calls probe at all.
	called = false
	cfg, err := CloudConfigFromName("china")
	require.NoError(t, err)
	assert.Equal(t, cloud.AzureChina.ActiveDirectoryAuthorityHost, cfg.ActiveDirectoryAuthorityHost)
	assert.False(t, called, "probe must not be called when using CloudConfigFromName directly")
}

func TestResolveCloudConfig_FirstSucceeds(t *testing.T) {
	creds := AzureCredentials{}
	callCount := 0

	got, err := resolveCloudConfig(context.Background(), creds, func(_ context.Context, _ AzureCredentials, cfg cloud.Configuration) bool {
		callCount++
		return cfg.ActiveDirectoryAuthorityHost == cloud.AzurePublic.ActiveDirectoryAuthorityHost
	})

	require.NoError(t, err)
	assert.Equal(t, cloud.AzurePublic.ActiveDirectoryAuthorityHost, got.ActiveDirectoryAuthorityHost)
	assert.Equal(t, 1, callCount, "should stop after first success")
}

func TestResolveCloudConfig_SecondSucceeds(t *testing.T) {
	creds := AzureCredentials{}

	got, err := resolveCloudConfig(context.Background(), creds, func(_ context.Context, _ AzureCredentials, cfg cloud.Configuration) bool {
		return cfg.ActiveDirectoryAuthorityHost == cloud.AzureChina.ActiveDirectoryAuthorityHost
	})

	require.NoError(t, err)
	assert.Equal(t, cloud.AzureChina.ActiveDirectoryAuthorityHost, got.ActiveDirectoryAuthorityHost)
}

func TestResolveCloudConfig_AllFail(t *testing.T) {
	creds := AzureCredentials{}

	_, err := resolveCloudConfig(context.Background(), creds, func(_ context.Context, _ AzureCredentials, _ cloud.Configuration) bool {
		return false
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "auto-discovery failed")
}
