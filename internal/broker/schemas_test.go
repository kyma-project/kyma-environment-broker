package broker

import (
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker/testutil"
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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"}, testutil.NewFakeConfigProvider(), "test-config-map")

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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"}, testutil.NewFakeConfigProvider(), "test-config-map")
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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"}, testutil.NewFakeConfigProvider(), "test-config-map")
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
