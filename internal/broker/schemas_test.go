package broker

import (
	"strings"
	"testing"

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
