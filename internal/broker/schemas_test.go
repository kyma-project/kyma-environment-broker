package broker

import (
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaService_validation(t *testing.T) {
	// Given
	plans, err := configuration.NewPlanSpecifications(strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1

`))
	require.NoError(t, err)
	providers, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
    regions:
       eu-central-1:
            displayName: "eu-central-1"
            zones: ["a", "b"]
       eu-west-1:
            displayName: "eu-west-1"
            zones: ["a", "b", "c"]
`))
	require.NoError(t, err)
	channelResolver := &fixture.FakeChannelResolver{}
	svc := NewSchemaService(providers, plans, nil, Config{}, StringList{"aws"}, channelResolver, nil)

	// When
	err = svc.Validate()

	// then
	assert.NoError(t, err)
}

func TestNewSchemaService_validation_MissingRegion(t *testing.T) {
	// Given
	plans, err := configuration.NewPlanSpecifications(strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1

`))
	require.NoError(t, err)
	providers, err := configuration.NewProviderSpec(strings.NewReader(`
aws:
    regions:
       eu-central-1:
            displayName: "eu-central-1"
            zones: ["a", "b"]
`))
	require.NoError(t, err)
	channelResolver := &fixture.FakeChannelResolver{}
	svc := NewSchemaService(providers, plans, nil, Config{}, StringList{"aws"}, channelResolver, nil)
	require.NoError(t, err)

	// When
	err = svc.Validate()

	// then
	assert.Error(t, err)
}

func TestNewSchemaService_validation_MissingProvider(t *testing.T) {
	// Given
	plans, err := configuration.NewPlanSpecifications(strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1

`))
	require.NoError(t, err)
	providers, err := configuration.NewProviderSpec(strings.NewReader(`
gcp:
    regions:
       eu-central-1:
            displayName: "eu-central-1"
            zones: ["a", "b"]
`))
	require.NoError(t, err)
	channelResolver := &fixture.FakeChannelResolver{}
	svc := NewSchemaService(providers, plans, nil, Config{}, StringList{"aws"}, channelResolver, nil)
	require.NoError(t, err)

	// When
	err = svc.Validate()

	// then
	assert.Error(t, err)
}

func TestSchemaPlans(t *testing.T) {
	// Given
	schemaService := createSchemaService(t)

	// When
	result := schemaService.Plans(PlansConfig{}, "cf-eu31", runtime.Azure)

	assert.True(t, *result[AzurePlanID].PlanUpdatable)
	assert.False(t, *result[BuildRuntimeAzurePlanID].PlanUpdatable)
}

func TestIsUpgradeable(t *testing.T) {
	// Given
	plansSpec, err := configuration.NewPlanSpecifications(strings.NewReader(`
aws,build-runtime-aws:
    upgradableToPlans:
        - build-runtime-aws
`))
	require.NoError(t, err)

	assert.True(t, plansSpec.IsUpgradable("aws"))
	assert.False(t, plansSpec.IsUpgradable("build-runtime-aws"))
}

func TestIsUpgradeable_EmptyUpgradeList(t *testing.T) {
	// Given
	plansSpec, err := configuration.NewPlanSpecifications(strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1
`))

	require.NoError(t, err)
	assert.False(t, plansSpec.IsUpgradable("aws"))
}

func TestMachineDisplayNames(t *testing.T) {
	providers, err := configuration.NewProviderSpec(strings.NewReader(`
azure:
  machines:
    Standard_D4s: Standard_D4s (4vCPU, 16GB RAM)
    Standard_D4s_v5: Standard_D4s_v5 (4vCPU, 16GB RAM) - use version-agnostic Standard_D4s
    Standard_D4_v3: Standard_D4_v3 (4vCPU, 16GB RAM) - use version-agnostic Standard_D4s
    Standard_F2s_v2: Standard_F2s_v2 (2vCPU, 4GB RAM)
    Standard_NC4as_T4_v3: Standard_NC4as_T4_v3 (1GPU, 4vCPU, 28GB RAM)*
    Standard_NC8as_T4_v3: Standard_NC4as_T4_v3 (1GPU, 4vCPU, 28GB RAM)* - test
  machinesVersions:
    Standard_D{size}s: Standard_D{size}s_v5
    Standard_D{size}s_v5: Standard_D{size}s_v5
    Standard_D{size}_v3: Standard_D{size}s_v5
`))
	require.NoError(t, err)

	volumeProvider := &fixedVolumeSizeProvider{
		sizes: map[runtime.CloudProvider]map[string]int{
			runtime.Azure: {
				"standard_d4s_v5":      80,
				"standard_f2s_v2":      80,
				"standard_nc4as_t4_v3": 80,
				"standard_nc8as_t4_v3": 86,
			},
		},
	}

	svc := NewSchemaService(providers, nil, nil, Config{}, StringList{}, &fixture.FakeChannelResolver{}, volumeProvider)

	// When
	result := svc.machineDisplayNames(runtime.Azure, []string{
		"Standard_D4s",
		"Standard_D4s_v5",
		"Standard_D4_v3",
		"Standard_F2s_v2",
		"Standard_NC4as_T4_v3",
		"Standard_NC8as_T4_v3",
	})

	// Then
	assert.Equal(t, "Standard_D4s (4vCPU, 16GB RAM, 80Gi volume)", result["Standard_D4s"])
	assert.Equal(t, "Standard_D4s_v5 (4vCPU, 16GB RAM, 80Gi volume) - use version-agnostic Standard_D4s", result["Standard_D4s_v5"])
	assert.Equal(t, "Standard_D4_v3 (4vCPU, 16GB RAM, 80Gi volume) - use version-agnostic Standard_D4s", result["Standard_D4_v3"])
	assert.Equal(t, "Standard_F2s_v2 (2vCPU, 4GB RAM, 80Gi volume)", result["Standard_F2s_v2"])
	assert.Equal(t, "Standard_NC4as_T4_v3 (1GPU, 4vCPU, 28GB RAM, 80Gi volume)*", result["Standard_NC4as_T4_v3"])
	assert.Equal(t, "Standard_NC4as_T4_v3 (1GPU, 4vCPU, 28GB RAM, 86Gi volume)* - test", result["Standard_NC8as_T4_v3"])
}
