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

func TestResolveCloudConfig_ExplicitName(t *testing.T) {
	// With an explicit name, ResolveCloudConfig must return the correct constant
	// without any network probe — credentials can be empty/invalid.
	creds := AzureCredentials{}

	got, err := ResolveCloudConfig(context.Background(), creds, "china")
	require.NoError(t, err)
	assert.Equal(t, cloud.AzureChina, got)

	got, err = ResolveCloudConfig(context.Background(), creds, "public")
	require.NoError(t, err)
	assert.Equal(t, cloud.AzurePublic, got)

	got, err = ResolveCloudConfig(context.Background(), creds, "usgov")
	require.NoError(t, err)
	assert.Equal(t, cloud.AzureGovernment, got)
}

func TestResolveCloudConfig_ExplicitName_Invalid(t *testing.T) {
	creds := AzureCredentials{}
	_, err := ResolveCloudConfig(context.Background(), creds, "invalid")
	require.Error(t, err)
}
