package rules

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestMatch_UseValidRuleset(t *testing.T) {
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

	defer func() { _ = os.Remove(tmpfile) }()

	svc, err := NewRulesServiceFromFile(tmpfile, sets.New[string]("azure", "gcp", "trial", "aws", "free"), sets.New[string]("azure", "gcp", "trial", "aws", "free"))
	require.NoError(t, err)

	require.NotNil(t, svc.ValidRules)

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
				RawData: RawData{
					Rule:   "azure(PR=cf-ch20) -> EU",
					RuleNo: 1,
				},
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
				RawData: RawData{
					Rule:   "aws(PR=cf-eu11) -> EU",
					RuleNo: 5,
				},
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
				RawData: RawData{
					Rule:   "free",
					RuleNo: 9,
				},
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
				RawData: RawData{
					Rule:   "gcp(PR=cf-sa30) -> PR,HR",
					RuleNo: 6,
				},
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
				RawData: RawData{
					Rule:   "gcp(PR=cf-sa30) -> PR,HR",
					RuleNo: 6,
				},
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
				RawData: RawData{
					Rule:   "trial -> S",
					RuleNo: 7,
				},
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
				RawData: RawData{
					Rule:   "trial(PR=cf-eu11) -> EU, S",
					RuleNo: 8,
				},
			},
		},
	} {
		t.Run(tn, func(t *testing.T) {

			result, found := svc.MatchProvisioningAttributesWithValidRuleset(&tc.given)
			assert.True(t, found)
			assert.Equal(t, tc.expected, result)
		})
	}

}
