package broker

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"

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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"})

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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"})
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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"})
	require.NoError(t, err)

	// When
	err = svc.Validate()

	// then
	assert.Error(t, err)
}

func TestNewSchemaService_validation_ZonesDiscoveryEnabledForNotAWSProvider(t *testing.T) {
	// Given
	plans, err := configuration.NewPlanSpecifications(strings.NewReader(`
gcp,build-runtime-gcp:
        regions:
            default:
                - europe-west3
`))
	require.NoError(t, err)
	providers, err := configuration.NewProviderSpec(strings.NewReader(`
gcp:
  zonesDiscovery: true
  regions:
    europe-west3:
      displayName: europe-west3 (Europe, Frankfurt)
`))
	require.NoError(t, err)
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"})

	// When
	err = svc.Validate()

	// then
	assert.EqualError(t, err, "zone discovery is not yet supported for the GCP provider")
}

func TestNewSchemaService_validation_ZonesDiscoveryEnabled(t *testing.T) {
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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"gcp"})

	// When
	err = svc.Validate()

	// then
	assert.NoError(t, err)
}

func TestNewSchemaService_validation_ZonesDiscoveryEnabledAndStaticZonesDefined(t *testing.T) {
	cw := &captureWriter{buf: &bytes.Buffer{}}
	handler := slog.NewTextHandler(cw, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)

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
	svc := NewSchemaService(providers, plans, nil, Config{}, EnablePlans{"aws"})

	// When
	err = svc.Validate()

	// then
	assert.NoError(t, err)

	logContents := cw.buf.String()
	assert.Contains(t, logContents, "Provider AWS (plan aws) has zones discovery enabled, but region eu-central-1 is configured with 2 static zones, which will be ignored.")
	assert.Contains(t, logContents, "Provider AWS (plan aws) has zones discovery enabled, but region eu-west-1 is configured with 3 static zones, which will be ignored.")
	assert.Contains(t, logContents, "Provider AWS (plan aws) has zones discovery enabled, but machine type g6 in region eu-central-1 is configured with 4 static zones, which will be ignored.")
	assert.Contains(t, logContents, "Provider AWS (plan build-runtime-aws) has zones discovery enabled, but region eu-central-1 is configured with 2 static zones, which will be ignored.")
	assert.Contains(t, logContents, "Provider AWS (plan build-runtime-aws) has zones discovery enabled, but region eu-west-1 is configured with 3 static zones, which will be ignored.")
	assert.Contains(t, logContents, "Provider AWS (plan build-runtime-aws) has zones discovery enabled, but machine type g6 in region eu-central-1 is configured with 4 static zones, which will be ignored.")
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

type captureWriter struct {
	buf *bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}
