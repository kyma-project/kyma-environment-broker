package configuration

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSpec(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: [ "a", "b", "f" ]
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
        zones: [ "a", "b", "c" ]
azure:
    regions:
      westeurope:
        displayName: "westeurope (Europe, Netherlands)"
        zones: [ "1", "2", "3" ]
`))
	require.NoError(t, err)

	// when / then

	assert.Equal(t, "eu-central-1 (Europe, Frankfurt)", providerSpec.RegionDisplayName(runtime.AWS, "eu-central-1"))
	assert.Equal(t, []string{"a", "b", "f"}, providerSpec.Zones(runtime.AWS, "eu-central-1"))

	assert.Equal(t, "westeurope (Europe, Netherlands)", providerSpec.RegionDisplayName(runtime.Azure, "westeurope"))
	assert.Equal(t, []string{"1", "2", "3"}, providerSpec.Zones(runtime.Azure, "westeurope"))
}

func TestProviderSpec_NotDefined(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: [ "a", "b", "f" ]
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
        zones: [ "a", "b", "c" ]
azure:
    regions:
      westeurope:
        displayName: "westeurope (Europe, Netherlands)"
        zones: [ "1", "2", "3" ]

`))
	require.NoError(t, err)

	// when / then

	assert.Equal(t, "us-east-1", providerSpec.RegionDisplayName(runtime.AWS, "us-east-1"))
	assert.Equal(t, []string{}, providerSpec.Zones(runtime.AWS, "us-east-1"))
}

func TestProviderSpec_Validation(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
  aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: []
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
      eu-west-1: 
        zones: [ "a", "b", "c" ]
      us-east-1:
        displayName: "us-east-1 (US, Virginia)"
        zones: [ "a", "b", "c" ]
`))
	require.NoError(t, err)

	// when / then

	assert.Errorf(t, providerSpec.Validate(runtime.AWS, "eu-central-1"), "region eu-central-1 for provider aws has no zones defined")
	assert.Errorf(t, providerSpec.Validate(runtime.AWS, "eu-west-2"), "region eu-west-2 for provider aws has no zones defined")
	assert.Errorf(t, providerSpec.Validate(runtime.AWS, "eu-west-1"), "region eu-west-1 for provider aws has no display name defined")
	assert.NoError(t, providerSpec.Validate(runtime.AWS, "us-east-1"))
}

func TestProviderSpec_ValidateZonesDiscovery(t *testing.T) {
	t.Run("should fail when zonesDiscovery enabled on nonAWS provider", func(t *testing.T) {
		// given
		providerSpec, err := NewProviderSpec(strings.NewReader(`
gcp:
  zonesDiscovery: true
`))
		require.NoError(t, err)

		// when / then
		err = providerSpec.ValidateZonesDiscovery()
		assert.EqualError(t, err, "zone discovery is not yet supported for the gcp provider")
	})

	t.Run("should pass when zonesDiscovery enabled on AWS provider", func(t *testing.T) {
		// given
		cw := &captureWriter{buf: &bytes.Buffer{}}
		handler := slog.NewTextHandler(cw, nil)
		logger := slog.New(handler)
		slog.SetDefault(logger)

		providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  zonesDiscovery: true
  regions:
    eu-central-1:
      displayName: "eu-central-1"
    eu-west-1:
      displayName: "eu-west-1"
  regionsSupportingMachine:
    g6:
      eu-central-1:
`))
		require.NoError(t, err)

		// when / then
		err = providerSpec.ValidateZonesDiscovery()
		assert.NoError(t, err)

		logContents := cw.buf.String()
		assert.Empty(t, logContents)
	})

	t.Run("should pass when zonesDiscovery enabled and static configuration provided on AWS provider", func(t *testing.T) {
		// given
		cw := &captureWriter{buf: &bytes.Buffer{}}
		handler := slog.NewTextHandler(cw, nil)
		logger := slog.New(handler)
		slog.SetDefault(logger)

		providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  zonesDiscovery: true
  regions:
    eu-central-1:
      displayName: "eu-central-1"
      zones: ["a", "b"]
    eu-west-1:
      displayName: "eu-west-1"
      zones: ["a", "b", "c"]
  regionsSupportingMachine:
    g6:
      eu-central-1: ["a", "b", "c", "d"]
`))
		require.NoError(t, err)

		// when / then
		err = providerSpec.ValidateZonesDiscovery()
		assert.NoError(t, err)

		logContents := cw.buf.String()
		assert.Contains(t, logContents, "Provider aws has zones discovery enabled, but region eu-central-1 is configured with 2 static zone(s), which will be ignored.")
		assert.Contains(t, logContents, "Provider aws has zones discovery enabled, but region eu-west-1 is configured with 3 static zone(s), which will be ignored.")
		assert.Contains(t, logContents, "Provider aws has zones discovery enabled, but machine type g6 in region eu-central-1 is configured with 4 static zone(s), which will be ignored.")
	})
}

func TestProviderSpec_ZonesDiscovery(t *testing.T) {
	tests := []struct {
		name       string
		inputYAML  string
		provider   runtime.CloudProvider
		wantResult bool
	}{
		{
			name: "zonesDiscovery true",
			inputYAML: `
  aws:
    zonesDiscovery: true
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: []
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
      eu-west-1: 
        zones: [ "a", "b", "c" ]
      us-east-1:
        displayName: "us-east-1 (US, Virginia)"
        zones: [ "a", "b", "c" ]
`,
			provider:   runtime.AWS,
			wantResult: true,
		},
		{
			name: "zonesDiscovery false",
			inputYAML: `
  aws:
    zonesDiscovery: false
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: []
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
      eu-west-1: 
        zones: [ "a", "b", "c" ]
      us-east-1:
        displayName: "us-east-1 (US, Virginia)"
        zones: [ "a", "b", "c" ]
`,
			provider:   runtime.AWS,
			wantResult: false,
		},
		{
			name: "zonesDiscovery missing field",
			inputYAML: `
  aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: []
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
      eu-west-1: 
        zones: [ "a", "b", "c" ]
      us-east-1:
        displayName: "us-east-1 (US, Virginia)"
        zones: [ "a", "b", "c" ]
`,
			provider:   runtime.AWS,
			wantResult: false,
		},
		{
			name: "zonesDiscovery missing provider",
			inputYAML: `
  aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
        zones: []
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
      eu-west-1: 
        zones: [ "a", "b", "c" ]
      us-east-1:
        displayName: "us-east-1 (US, Virginia)"
        zones: [ "a", "b", "c" ]
`,
			provider:   runtime.GCP,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerSpec, err := NewProviderSpec(strings.NewReader(tt.inputYAML))
			require.NoError(t, err)

			got := providerSpec.ZonesDiscovery(tt.provider)
			assert.Equal(t, tt.wantResult, got)
		})
	}
}

func TestProviderSpec_Regions(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
    regions:
      eu-central-1:
        displayName: "eu-central-1 (Europe, Frankfurt)"
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
`))
	require.NoError(t, err)

	// when / then

	regions := providerSpec.Regions(runtime.AWS)
	assert.Equal(t, []string{"eu-central-1", "eu-west-2"}, regions)
}

func TestProviderSpec_MachineTypes(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  machines:
    "m6i.large": "m6i.large (2vCPU, 8GB RAM)"
    "g6.xlarge": "g6.xlarge (1GPU, 4vCPU, 16GB RAM)*"
    "g4dn.xlarge": "g4dn.xlarge (1GPU, 4vCPU, 16GB RAM)*"
`))
	require.NoError(t, err)

	// when / then

	machineTypes := providerSpec.MachineTypes(runtime.AWS)
	assert.ElementsMatch(t, []string{"m6i.large", "g6.xlarge", "g4dn.xlarge"}, machineTypes)
}

type captureWriter struct {
	buf *bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}
