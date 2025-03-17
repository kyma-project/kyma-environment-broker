package rules

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/require"
)

func TestMatchDifferentArtificialScenarios(t *testing.T) {
	//content := `rule:
	//- aws                             # pool: hyperscalerType: aws
	//- aws(PR=cf-eu11) -> EU           # pool: hyperscalerType: aws_cf-eu11; euAccess: true
	//- azure                           # pool: hyperscalerType: azure
	//- azure(PR=cf-ch20) -> EU         # pool: hyperscalerType: azure; euAccess: true
	//- gcp                             # pool: hyperscalerType: gcp
	//- gcp(PR=cf-sa30)                 # pool: hyperscalerType: gcp_cf-sa30
	//- trial -> S                      # pool: hyperscalerType: azure; shared: true - TRIAL POOL`

	content := `rule:
  - azure(PR=cf-ch20) -> EU
  - gcp
  - azure
  - aws
  - aws(PR=cf-eu11) -> EU
  - gcp(PR=cf-sa30) -> PR,HR        # HR must be taken from ProvisioningAttributes
  - trial -> S					  # TRIAL POOL
  - trial(PR=cf-eu11) -> EU, S
  - free
`

	tmpfile, err := CreateTempFile(content)
	require.NoError(t, err)

	defer os.Remove(tmpfile)

	svc, err := NewRulesServiceFromFile(tmpfile, &broker.EnablePlans{"azure", "gcp", "trial", "aws", "free"})
	require.NoError(t, err)

	for _, result := range svc.Parsed.Results {
		if result.HasErrors() {
			fmt.Println(result.ParsingErrors)
			fmt.Println(result.ProcessingErrors)
		}
	}

	for tn, tc := range map[string]struct {
		given    ProvisioningAttributes
		expected Result
	}{
		"azure eu": {
			given: ProvisioningAttributes{
				Plan:              "azure",
				PlatformRegion:    "cf-ch20",
				HyperscalerRegion: "switzerlandnorth",
				Hyperscaler:       "azure",
			},
			expected: Result{
				HyperscalerType: "azure",
				EUAccess:        true,
				Shared:          false,
			},
		},
		"aws eu": {
			given: ProvisioningAttributes{
				Plan:              "aws",
				PlatformRegion:    "cf-eu11",
				HyperscalerRegion: "eu-central1",
				Hyperscaler:       "aws",
			},
			expected: Result{
				HyperscalerType: "aws",
				EUAccess:        true,
				Shared:          false,
			},
		},
		"free": {
			given: ProvisioningAttributes{
				Plan:              "free",
				PlatformRegion:    "cf-eu21",
				HyperscalerRegion: "westeurope",
				Hyperscaler:       "azure",
			},
			expected: Result{
				HyperscalerType: "azure",
				EUAccess:        false,
				Shared:          false,
			},
		},
		"gcp with PR and HR in labels": {
			given: ProvisioningAttributes{
				Plan:              "gcp",
				PlatformRegion:    "cf-sa30",
				HyperscalerRegion: "ksa",
				Hyperscaler:       "gcp",
			},
			expected: Result{
				HyperscalerType: "gcp_cf-sa30_ksa",
				EUAccess:        false,
				Shared:          false,
			},
		},
		// second check to verify idempotence
		"gcp with PR and HR in labels2": {
			given: ProvisioningAttributes{
				Plan:              "gcp",
				PlatformRegion:    "cf-sa30",
				HyperscalerRegion: "ksa",
				Hyperscaler:       "gcp",
			},
			expected: Result{
				HyperscalerType: "gcp_cf-sa30_ksa",
				EUAccess:        false,
				Shared:          false,
			},
		},
		"trial": {
			given: ProvisioningAttributes{
				Plan:              "trial",
				PlatformRegion:    "cf-us11",
				HyperscalerRegion: "us-west",
				Hyperscaler:       "aws",
			},
			expected: Result{
				HyperscalerType: "aws",
				EUAccess:        false,
				Shared:          true,
			},
		},
		"trial eu": {
			given: ProvisioningAttributes{
				Plan:              "trial",
				PlatformRegion:    "cf-eu11",
				HyperscalerRegion: "us-west",
				Hyperscaler:       "aws",
			},
			expected: Result{
				HyperscalerType: "aws",
				EUAccess:        true,
				Shared:          true,
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {

			result, found := svc.MatchProvisioningAttributes(&tc.given)
			assert.True(t, found)
			assert.Equal(t, tc.expected, result)
		})
	}

}
