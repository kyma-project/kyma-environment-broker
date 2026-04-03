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

func TestIsRegionSupported(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  regionsSupportingMachine:
    m8g:
      ap-northeast-1: [a, b, c, d]
      ap-southeast-1: []
      ca-central-1: null
    r8i:
      ap-northeast-1: [a, b, c, d]
  machinesVersions:
    "ri.{size}": "r8i.{size}"
gcp:
  regionsSupportingMachine:
    c2d-highmem:
      us-central1: null
      southamerica-east1: [a, b, c]
azure:
  regionsSupportingMachine:
    Standard_L:
      uksouth: [a]
      japaneast: null
      brazilsouth: [a, b]
`))
	require.NoError(t, err)

	tests := []struct {
		name          string
		cloudProvider runtime.CloudProvider
		region        string
		machineType   string
		expected      bool
	}{
		{"Supported m8g", runtime.AWS, "ap-northeast-1", "m8g.large", true},
		{"Unsupported m8g", runtime.AWS, "us-central1", "m8g.2xlarge", false},
		{"Supported ri", runtime.AWS, "ap-northeast-1", "ri.large", true},
		{"Unsupported ri", runtime.AWS, "us-central1", "ri.large", false},
		{"Supported c2d-highmem", runtime.GCP, "us-central1", "c2d-highmem-32", true},
		{"Unsupported c2d-highmem", runtime.GCP, "ap-southeast-1", "c2d-highmem-64", false},
		{"Supported Standard_L", runtime.Azure, "uksouth", "Standard_L8s_v3", true},
		{"Unsupported Standard_L", runtime.Azure, "us-west", "Standard_L48s_v3", false},
		{"Unknown machine type defaults to true", runtime.Azure, "any-region", "unknown-type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result := providerSpec.IsRegionSupported(tt.cloudProvider, tt.region, tt.machineType)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  regionsSupportingMachine:
    m8g:
      ap-northeast-1: [a, b, c, d]
      ap-southeast-1: []
      ca-central-1: null
    r8i:
      ap-northeast-1: [a, b, c, d]
  machinesVersions:
    "ri.{size}": "r8i.{size}"
gcp:
  regionsSupportingMachine:
    c2d-highmem:
      us-central1: null
      southamerica-east1: [a, b, c]
azure:
  regionsSupportingMachine:
    Standard_L:
      uksouth: [a]
      japaneast: null
      brazilsouth: [a, b]
`))
	require.NoError(t, err)

	tests := []struct {
		name          string
		cloudProvider runtime.CloudProvider
		machineType   string
		expected      []string
	}{
		{"Supported m8g", runtime.AWS, "m8g.large", []string{"ap-northeast-1", "ap-southeast-1", "ca-central-1"}},
		{"Supported ri", runtime.AWS, "ri.large", []string{"ap-northeast-1"}},
		{"Supported c2d-highmem", runtime.GCP, "c2d-highmem-32", []string{"southamerica-east1", "us-central1"}},
		{"Supported Standard_L", runtime.Azure, "Standard_L8s_v3", []string{"brazilsouth", "japaneast", "uksouth"}},
		{"Unknown machine type", runtime.Azure, "unknown-type", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result := providerSpec.SupportedRegions(tt.cloudProvider, tt.machineType)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAvailableZones(t *testing.T) {
	// given
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  regionsSupportingMachine:
    m8g:
      ap-northeast-1: [a, b, c, d]
      ap-southeast-1: []
      ca-central-1: null
    r8i:
      ap-northeast-1: [a, b, c, d]
  machinesVersions:
    "ri.{size}": "r8i.{size}"
gcp:
  regionsSupportingMachine:
    c2d-highmem:
      us-central1: null
      southamerica-east1: [a, b, c]
azure:
  regionsSupportingMachine:
    Standard_L:
      uksouth: [a]
      japaneast: null
      brazilsouth: [a, b]
`))
	require.NoError(t, err)

	tests := []struct {
		name          string
		cloudProvider runtime.CloudProvider
		machineType   string
		region        string
		expected      []string
	}{
		{
			name:          "AWS - region with 4 zones",
			cloudProvider: runtime.AWS,
			machineType:   "m8g.large",
			region:        "ap-northeast-1",
			expected:      []string{"a", "b", "c", "d"},
		},
		{
			name:          "AWS - region with empty zones list",
			cloudProvider: runtime.AWS,
			machineType:   "m8g.large",
			region:        "ap-southeast-1",
			expected:      []string{},
		},
		{
			name:          "AWS - region with null zones",
			cloudProvider: runtime.AWS,
			machineType:   "m8g.large",
			region:        "ca-central-1",
			expected:      []string{},
		},
		{
			name:          "AWS - region with 4 zones (version-agnostic machine type)",
			cloudProvider: runtime.AWS,
			machineType:   "ri.large",
			region:        "ap-northeast-1",
			expected:      []string{"a", "b", "c", "d"},
		},
		{
			name:          "GCP - region with null zones",
			cloudProvider: runtime.GCP,
			machineType:   "c2d-highmem-8",
			region:        "us-central1",
			expected:      []string{},
		},
		{
			name:          "GCP - region with 3 zones",
			cloudProvider: runtime.GCP,
			machineType:   "c2d-highmem-8",
			region:        "southamerica-east1",
			expected:      []string{"a", "b", "c"},
		},
		{
			name:          "Azure - region with 1 zone",
			cloudProvider: runtime.Azure,
			machineType:   "Standard_L8s_v3",
			region:        "uksouth",
			expected:      []string{"a"},
		},
		{
			name:          "Azure - region with null zones",
			cloudProvider: runtime.Azure,
			machineType:   "Standard_L8s_v3",
			region:        "japaneast",
			expected:      []string{},
		},
		{
			name:          "Azure - region with 2 zones",
			cloudProvider: runtime.Azure,
			machineType:   "Standard_L8s_v3",
			region:        "brazilsouth",
			expected:      []string{"a", "b"},
		},
		{
			name:          "Azure - not supported region",
			cloudProvider: runtime.Azure,
			machineType:   "Standard_L8s_v3",
			region:        "notSupportedRegion",
			expected:      []string{},
		},
		{
			name:          "Azure - not supported machine type",
			cloudProvider: runtime.Azure,
			machineType:   "notSupportedMachineType",
			region:        "notSupportedRegion",
			expected:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			result := providerSpec.AvailableZones(tt.cloudProvider, tt.machineType, tt.region)

			// then
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestProviderSpec_ValidateMachinesVersions(t *testing.T) {
	providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  machinesVersions:
    "x.{size}{c_size}": "x1.{size}{c_size}"
    "dup.{size}.{size}": "dup2.{size}"
    "broken.{size": "broken.{size}"
    "bad-output.{size}": "bad.{other}"
`))
	require.NoError(t, err)

	err = providerSpec.ValidateMachinesVersions()
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, `provider "aws": invalid input template "x.{size}{c_size}": adjacent placeholders are not allowed`)
	assert.Contains(t, msg, `provider "aws": invalid input template "dup.{size}.{size}": duplicate placeholder "{size}"`)
	assert.Contains(t, msg, `provider "aws": invalid input template "broken.{size": unclosed placeholder starting at position 7`)
	assert.Contains(t, msg, `provider "aws": invalid mapping "bad-output.{size}" -> "bad.{other}": output placeholder "{other}" is not defined in input template`)
}

func TestProviderSpec_ResolveMachineType(t *testing.T) {
	t.Run("AWS", func(t *testing.T) {
		providerSpec, err := NewProviderSpec(strings.NewReader(`
aws:
  machinesVersions:
    "mi.{size}": "m6i.{size}"
    "ci.{size}": "c7i.{size}"
    "g.{size}": "g6.{size}"
    "gdn.{size}": "g4dn.{size}"
    "ri.{size}": "r8i.{size}"
    "ii.{size}": "i7i.{size}"
    "m5.{size}": "m6i.{size}"
`))
		require.NoError(t, err)

		tests := map[string]string{
			"mi.large":     "m6i.large",
			"ci.xlarge":    "c7i.xlarge",
			"g.2xlarge":    "g6.2xlarge",
			"gdn.4xlarge":  "g4dn.4xlarge",
			"ri.8xlarge":   "r8i.8xlarge",
			"ii.12xlarge":  "i7i.12xlarge",
			"m6i.16xlarge": "m6i.16xlarge",
			"m5.large":     "m6i.large",
			"c7i.xlarge":   "c7i.xlarge",
			"g6.2xlarge":   "g6.2xlarge",
			"g4dn.4xlarge": "g4dn.4xlarge",
		}

		for input, expected := range tests {
			t.Run(input, func(t *testing.T) {
				actual := providerSpec.ResolveMachineType(runtime.AWS, input)
				assert.Equal(t, expected, actual)
			})
		}
	})

	t.Run("Azure", func(t *testing.T) {
		providerSpec, err := NewProviderSpec(strings.NewReader(`
azure:
  machinesVersions:
    Standard_D{size}s: Standard_D{size}s_v5
    Standard_D{size}: Standard_D{size}_v3
    Standard_F{size}s: Standard_F{size}s_v2
    Standard_NC{size}as_T4: Standard_NC{size}as_T4_v3
    Standard_E{size}s: Standard_E{size}s_v6
    Standard_L{size}s: Standard_L{size}s_v3
`))
		require.NoError(t, err)

		tests := map[string]string{
			"Standard_D2s":         "Standard_D2s_v5",
			"Standard_D4":          "Standard_D4_v3",
			"Standard_F8s":         "Standard_F8s_v2",
			"Standard_NC16as_T4":   "Standard_NC16as_T4_v3",
			"Standard_E20s":        "Standard_E20s_v6",
			"Standard_L32s":        "Standard_L32s_v3",
			"Standard_D48s_v5":     "Standard_D48s_v5",
			"Standard_D64_v3":      "Standard_D64_v3",
			"Standard_F2s_v2":      "Standard_F2s_v2",
			"Standard_NC4as_T4_v3": "Standard_NC4as_T4_v3",
		}

		for input, expected := range tests {
			t.Run(input, func(t *testing.T) {
				actual := providerSpec.ResolveMachineType(runtime.Azure, input)
				assert.Equal(t, expected, actual)
			})
		}
	})

	t.Run("GCP", func(t *testing.T) {
		providerSpec, err := NewProviderSpec(strings.NewReader(`
gcp:
  machinesVersions:
    n-standard-{size}: n2-standard-{size}
    cd-highcpu-{size}: c2d-highcpu-{size}
    g-standard-{size}: g2-standard-{size}
    m-ultramem-{size}: m3-ultramem-{size}
    z-highmem-{size}: z3-highmem-{size}-standardlssd
`))
		require.NoError(t, err)

		tests := map[string]string{
			"n-standard-2":   "n2-standard-2",
			"cd-highcpu-4":   "c2d-highcpu-4",
			"g-standard-8":   "g2-standard-8",
			"m-ultramem-32":  "m3-ultramem-32",
			"z-highmem-44":   "z3-highmem-44-standardlssd",
			"n2-standard-48": "n2-standard-48",
			"c2d-highcpu-56": "c2d-highcpu-56",
			"g2-standard-4":  "g2-standard-4",
		}

		for input, expected := range tests {
			t.Run(input, func(t *testing.T) {
				actual := providerSpec.ResolveMachineType(runtime.GCP, input)
				assert.Equal(t, expected, actual)
			})
		}
	})

	t.Run("SAP Cloud Infrastructure", func(t *testing.T) {
		providerSpec, err := NewProviderSpec(strings.NewReader(`
sap-converged-cloud:
  machinesVersions:
    g_c{c_size}_m{m_size}: g_c{c_size}_m{m_size}_v2
`))
		require.NoError(t, err)

		tests := map[string]string{
			"g_c2_m8":    "g_c2_m8_v2",
			"g_c64_m256": "g_c64_m256_v2",
		}

		for input, expected := range tests {
			t.Run(input, func(t *testing.T) {
				actual := providerSpec.ResolveMachineType(runtime.SapConvergedCloud, input)
				assert.Equal(t, expected, actual)
			})
		}
	})

	t.Run("Alibaba Cloud", func(t *testing.T) {
		providerSpec, err := NewProviderSpec(strings.NewReader(`
alicloud:
  machinesVersions:
    ecs.gi.{size}: ecs.g9i.{size}
`))
		require.NoError(t, err)

		tests := map[string]string{
			"ecs.gi.large":     "ecs.g9i.large",
			"ecs.g9i.16xlarge": "ecs.g9i.16xlarge",
		}

		for input, expected := range tests {
			t.Run(input, func(t *testing.T) {
				actual := providerSpec.ResolveMachineType(runtime.Alicloud, input)
				assert.Equal(t, expected, actual)
			})
		}
	})
}

type captureWriter struct {
	buf *bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}
