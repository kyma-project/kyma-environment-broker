package rules

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

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
  - gcp(PR=cf-sa30) -> PR,HR        # HR must be taken from ProvisioningAttributes
  - trial -> S					  # TRIAL POOL
  - trial(PR=cf-eu11) -> EU, S
`

	tmpfile, err := CreateTempFile(content)
	require.NoError(t, err)

	defer os.Remove(tmpfile)

	svc, err := NewRulesServiceFromFile(tmpfile, &broker.EnablePlans{"azure", "gcp", "trial", "aws"}, false, false, true)
	require.NoError(t, err)

	for _, result := range svc.Parsed.Results {
		if result.HasErrors() {
			fmt.Println(result.ParsingErrors)
			fmt.Println(result.ProcessingErrors)
		}
	}

	for tn, tc := range map[string]struct {
		given    ProvisioningAttributes
		expected map[string]string
	}{
		"azure eu": {
			given: ProvisioningAttributes{
				Plan:              "azure",
				PlatformRegion:    "cf-ch20",
				HyperscalerRegion: "switzerlandnorth",
			},
			expected: map[string]string{
				"hyperscalerType": "azure",
				"euAccess":        "true",
			},
		},
		"gcp with PR and HR in labels": {
			given: ProvisioningAttributes{
				Plan:              "gcp",
				PlatformRegion:    "cf-sa30",
				HyperscalerRegion: "ksa",
			},
			expected: map[string]string{
				"hyperscalerType": "gcp_cf-sa30_ksa",
			},
		},
		"trial": {
			given: ProvisioningAttributes{
				Plan:              "trial",
				PlatformRegion:    "cf-us11",
				HyperscalerRegion: "us-west",
			},
			expected: map[string]string{
				"hyperscalerType": "aws",
				"shared":          "true",
			},
		},
		"trial eu": {
			given: ProvisioningAttributes{
				Plan:              "trial",
				PlatformRegion:    "cf-eu11",
				HyperscalerRegion: "us-west",
			},
			expected: map[string]string{
				"hyperscalerType": "aws",
				"shared":          "true",
				"euAccess":        "true",
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			result := svc.Match(&tc.given)

			require.NoError(t, err)

			/**

			  HAP step

			  svc.Match(ProvisioningAttr) -> one result


			  result.Labels()

			  result.IsShared()


			*/

			//found := false
			for _, matchingResult := range result {
				if matchingResult.FinalMatch {
					fmt.Println("Given:")
					fmt.Println("   ", tc)
					fmt.Println("Matched:")
					for k, v := range matchingResult.Labels() {
						fmt.Println(k, v)
					}
					assert.Equal(t, tc.expected, matchingResult.Labels())
					//require.Equal(t, "azure", matchingResult.Rule.Plan)
					//require.Equal(t, "", matchingResult.Rule.HyperscalerRegion)
					//require.Equal(t, "", matchingResult.Rule.PlatformRegion)
					//found = true
				}
			}
		})
	}

}
