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
				PatternAttribute{literal: "eu-west-2"}, false, false, false, false, 0,
				RawData{"", 44}},
			expectedKey: "aws(PR=cf-eu10,HR=eu-west-2)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, true, true, true, true, 44,
				RawData{"", 44}},
			expectedKey: "aws(PR=cf-eu10,HR=eu-west-2)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "eu-west-2"}, true, true, true, true, 44,
				RawData{"", 44}},
			expectedKey: "aws(PR=,HR=eu-west-2)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "", matchAny: true}, true, true, true, true, 44,
				RawData{"", 44}},
			expectedKey: "aws(PR=,HR=)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "azure"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "", matchAny: true}, true, true, true, true, 44,
				RawData{"", 44}},
			expectedKey: "azure(PR=,HR=)",
		},
		{
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "", matchAny: true}, true, true, true, true, 44,
				RawData{"", 44}},
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
		raw    RawData
		output *ValidRule
	}{
		{name: "simple aws",
			rule: &Rule{
				Plan:              "aws",
				PlatformRegion:    "",
				HyperscalerRegion: "",
			},
			raw: RawData{"aws", 44},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "", matchAny: true},
				HyperscalerRegion:       PatternAttribute{literal: "", matchAny: true},
				PlatformRegionSuffix:    false,
				HyperscalerRegionSuffix: false,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           2,
				RawData:                 RawData{"aws", 44},
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
			raw: RawData{"aws", 44},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "", matchAny: true},
				HyperscalerRegion:       PatternAttribute{literal: "", matchAny: true},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                true,
				Shared:                  true,
				MatchAnyCount:           2,
				RawData:                 RawData{"aws", 44},
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
			raw: RawData{"aws(PR=cf-eu10)", 44},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "cf-eu10", matchAny: false},
				HyperscalerRegion:       PatternAttribute{literal: "", matchAny: true},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           1,
				RawData:                 RawData{"aws(PR=cf-eu10)", 44},
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
			raw: RawData{"aws(HR=eu-west-2)", 44},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "", matchAny: true},
				HyperscalerRegion:       PatternAttribute{literal: "eu-west-2", matchAny: false},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           1,
				RawData:                 RawData{"aws(HR=eu-west-2)", 44},
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
			raw: RawData{"aws(HR=eu-west-2,PR=cf-eu10)", 44},
			output: &ValidRule{
				Plan:                    PatternAttribute{literal: "aws"},
				PlatformRegion:          PatternAttribute{literal: "cf-eu10", matchAny: false},
				HyperscalerRegion:       PatternAttribute{literal: "eu-west-2", matchAny: false},
				PlatformRegionSuffix:    true,
				HyperscalerRegionSuffix: true,
				EuAccess:                false,
				Shared:                  false,
				MatchAnyCount:           0,
				RawData:                 RawData{"aws(HR=eu-west-2,PR=cf-eu10)", 44},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			vr := toValidRule(tc.rule, tc.raw.Rule, tc.raw.RuleNo)
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
			name: "simple trial",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, false, false, false, 0,
				RawData{"trial(PR=cf-eu10, HR=eu-west-2)", 0}},
			expected: Result{
				HyperscalerType: "aws",
				EUAccess:        false,
				Shared:          false,
				RawData: RawData{
					Rule:   "trial(PR=cf-eu10, HR=eu-west-2)",
					RuleNo: 0,
				},
			},
		},
		{
			name: "trial with all suffixes",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, true, true, true, 0,
				RawData{"trial(PR=cf-eu10, HR=eu-west-2)->EU,PR,HR", 0}},
			expected: Result{
				HyperscalerType: "aws_cf-eu10_eu-west-2",
				EUAccess:        true,
				Shared:          false,
				RawData: RawData{
					Rule:   "trial(PR=cf-eu10, HR=eu-west-2)->EU,PR,HR",
					RuleNo: 0,
				},
			},
		},
		{
			name: "trial with platform region suffix only",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, true, true, false, 0,
				RawData{"trial(PR=cf-eu10, HR=eu-west-2)->EU,PR", 0}},
			expected: Result{
				HyperscalerType: "aws_cf-eu10",
				EUAccess:        true,
				Shared:          false,
				RawData: RawData{
					Rule:   "trial(PR=cf-eu10, HR=eu-west-2)->EU,PR",
					RuleNo: 0,
				},
			},
		},
		{
			name: "trial with hyperscaler region suffix only",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, true, true, false, true, 44,
				RawData{"trial(PR=cf-eu10, HR=eu-west-2)->EU,HR", 0}},
			expected: Result{
				HyperscalerType: "aws_eu-west-2",
				EUAccess:        true,
				Shared:          true,
				RawData: RawData{
					Rule:   "trial(PR=cf-eu10, HR=eu-west-2)->EU,HR",
					RuleNo: 0,
				},
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

func TestValidRule_Match(t *testing.T) {
	testCases := []struct {
		name     string
		input    *ValidRule
		expected bool
	}{
		{
			name: "specific trial",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, false, false, false, 0, RawData{}},
			expected: true,
		},
		{
			name: "general trial",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "", matchAny: true}, false, false, false, false, 0, RawData{}},
			expected: true,
		},
		{
			name: "plan mismatch",
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-2"}, false, false, false, false, 0, RawData{}},
			expected: false,
		},
		{
			name: "plan mismatch",
			input: &ValidRule{PatternAttribute{literal: "aws"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "", matchAny: true}, false, false, false, false, 0, RawData{}},
			expected: false,
		},
		{
			name: "hyperscaler region mismatch",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-eu10"},
				PatternAttribute{literal: "eu-west-1"}, false, false, false, false, 0, RawData{}},
			expected: false,
		},
		{
			name: "hyperscaler region mismatch",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "", matchAny: true},
				PatternAttribute{literal: "eu-west-1"}, false, false, false, false, 0, RawData{}},
			expected: false,
		},
		{
			name: "platform region mismatch",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-jp30"},
				PatternAttribute{literal: "eu-west-2"},
				false, false, false, false, 0, RawData{}},
			expected: false,
		},
		{
			name: "platform region mismatch",
			input: &ValidRule{PatternAttribute{literal: "trial"},
				PatternAttribute{literal: "cf-jp30"},
				PatternAttribute{literal: "", matchAny: true},
				false, false, false, false, 0, RawData{}},
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			result := tc.input.Match(&ProvisioningAttributes{Plan: "trial", Hyperscaler: "aws", PlatformRegion: "cf-eu10", HyperscalerRegion: "eu-west-2"})
			//then
			assert.Equal(t, tc.expected, result)
		})
	}
}
