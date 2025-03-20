package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidRule_keyString(t *testing.T) {
	testCases := []struct {
		input       *ValidRule
		expectedKey string
	}{
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, false, false, false, 0},
			expectedKey: "aws(PR=cf-eu10,HR=eu-west-2)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, true, true, true, true, 44},
			expectedKey: "aws(PR=cf-eu10,HR=eu-west-2)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "eu-west-2"}, true, true, true, true, 44},
			expectedKey: "aws(PR=,HR=eu-west-2)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "", matchAny: true}, true, true, true, true, 44},
			expectedKey: "aws(PR=,HR=)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "azure"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "", matchAny: true}, true, true, true, true, 44},
			expectedKey: "azure(PR=,HR=)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "", matchAny: true}, true, true, true, true, 44},
			expectedKey: "aws(PR=cf-eu10,HR=)",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.expectedKey, func(t *testing.T) {
			//then
			assert.Equal(t, tc.expectedKey, tc.input.keyString())
		})
	}

}

func TestRuleToValidRuleConversion(t *testing.T) {
	testCases := []struct {
		name   string
		rule   *Rule
		output *ValidRule
	}{
		{name: "simple aws",
			rule: &Rule{
				Plan:              "aws",
				PlatformRegion:    "",
				HyperscalerRegion: "",
			},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "", matchAny: true},
				HyperscalerRegion:       PatternAttribute{literal: "", matchAny: true},
				PlatformRegionSuffix:    false,
				HyperscalerRegionSuffix: false,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           2,
			},
		},
		{name: "aws with full right side",
			rule: &Rule{
				Plan:                    "aws",
				PlatformRegion:          "",
				HyperscalerRegion:       "",
				Shared:                  true,
				EuAccess:                true,
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
			},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "", matchAny: true},
				HyperscalerRegion:       PatternAttribute{literal: "", matchAny: true},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                true,
				Shared:                  true,
				MatchAnyCount:           2,
			},
		},
		{name: "aws with one literal",
			rule: &Rule{
				Plan:                    "aws",
				PlatformRegion:          "cf-eu10",
				HyperscalerRegion:       "",
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
			},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "cf-eu10", matchAny: false},
				HyperscalerRegion:       PatternAttribute{literal: "", matchAny: true},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           1,
			},
		},
		{name: "aws with second literal",
			rule: &Rule{
				Plan:                    "aws",
				PlatformRegion:          "",
				HyperscalerRegion:       "eu-west-2",
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
			},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "", matchAny: true},
				HyperscalerRegion:       PatternAttribute{literal: "eu-west-2", matchAny: false},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           1,
			},
		},
		{name: "aws with two literals",
			rule: &Rule{
				Plan:                    "aws",
				PlatformRegion:          "cf-eu10",
				HyperscalerRegion:       "eu-west-2",
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
			},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "cf-eu10", matchAny: false},
				HyperscalerRegion:       PatternAttribute{literal: "eu-west-2", matchAny: false},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			vr := toValidRule(tc.rule)
			//then
			assert.Equal(t, vr, tc.output)
		})
	}
}

func TestValidRule_toResult(t *testing.T) {
	testCases := []struct {
		name     string
		input    *ValidRule
		expected Result
	}{
		{
			name: "simple trial with aws",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, false, false, false, 0},
			expected: Result{
				HyperscalerType: "aws",
				EUAccess:        false,
				Shared:          false,
			},
		},
		{
			name: "trial with full right side",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, true, true, true, 0},
			expected: Result{
				HyperscalerType: "aws_cf-eu10_eu-west-2",
				EUAccess:        true,
				Shared:          false,
			},
		},
		{
			name: "trial with platform region only",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, true, true, false, 0},
			expected: Result{
				HyperscalerType: "aws_cf-eu10",
				EUAccess:        true,
				Shared:          false,
			},
		},
		{
			name: "trial with hyperscaler region only",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, true, true, false, true, 44},
			expected: Result{
				HyperscalerType: "aws_eu-west-2",
				EUAccess:        true,
				Shared:          true,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			result := tc.input.toResult(&ProvisioningAttributes{Plan: "trial", Hyperscaler: "aws", PlatformRegion: "cf-eu10", HyperscalerRegion: "eu-west-2"})
			//then
			assert.Equal(t, tc.expected, result)
		})
	}

}
