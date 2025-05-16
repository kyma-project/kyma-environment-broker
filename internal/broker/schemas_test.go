package broker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaService_validation(t *testing.T) {
	// Given
	plans := strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1

`)
	providers := strings.NewReader(`
aws:
    regions:
       eu-central-1:
            displayName: "eu-central-1"
            zones: ["a", "b"]
       eu-west-1:
            displayName: "eu-west-1"
            zones: ["a", "b", "c"]
`)
	svc, err := NewSchemaService(providers, plans, nil, Config{}, true, EnablePlans{"aws"})
	require.NoError(t, err)

	// When
	err = svc.Validate()

	// then
	assert.NoError(t, err)
}

func TestNewSchemaService_validation_MissingRegion(t *testing.T) {
	// Given
	plans := strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1

`)
	providers := strings.NewReader(`
aws:
    regions:
       eu-central-1:
            displayName: "eu-central-1"
            zones: ["a", "b"]
`)
	svc, err := NewSchemaService(providers, plans, nil, Config{}, true, EnablePlans{"aws"})
	require.NoError(t, err)

	// When
	err = svc.Validate()

	// then
	assert.Error(t, err)
}

func TestNewSchemaService_validation_MissingProvider(t *testing.T) {
	// Given
	plans := strings.NewReader(`
aws,build-runtime-aws:
        regions:
            cf-eu11:
                - eu-central-1
            default:
                - eu-west-1

`)
	providers := strings.NewReader(`
gcp:
    regions:
       eu-central-1:
            displayName: "eu-central-1"
            zones: ["a", "b"]
`)
	svc, err := NewSchemaService(providers, plans, nil, Config{}, true, EnablePlans{"aws"})
	require.NoError(t, err)

	// When
	err = svc.Validate()

	// then
	assert.Error(t, err)
}
