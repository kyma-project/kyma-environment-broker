package rules

import (
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRulesServiceFromFile(t *testing.T) {
	t.Run("should create RulesService from valid file ane parse simple rules", func(t *testing.T) {
		// given
		content := `rule:
                      - rule1
                      - rule2`

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)

		defer os.Remove(tmpfile)

		// when
		enabledPlans := &broker.EnablePlans{"rule1", "rule2"}
		service, err := NewRulesServiceFromFile(tmpfile, enabledPlans)

		// then
		require.NoError(t, err)
		require.NotNil(t, service)

		require.Equal(t, 2, len(service.ParsedRuleset.Results))
		for _, result := range service.ParsedRuleset.Results {
			require.False(t, result.HasErrors())
		}
	})

	t.Run("should return error when file path is empty", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("", &broker.EnablePlans{})

		// then
		require.Error(t, err)
		require.Nil(t, service)
		require.Equal(t, "No HAP rules file path provided", err.Error())
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("nonexistent.yaml", &broker.EnablePlans{})

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

	t.Run("should return error when YAML file is corrupted", func(t *testing.T) {
		// given
		content := "corrupted_content"

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)
		defer os.Remove(tmpfile)

		// when
		service, err := NewRulesServiceFromFile(tmpfile, &broker.EnablePlans{})

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

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

func TestPostParse(t *testing.T) {
	testCases := []struct {
		name               string
		inputRuleset       []string
		outputRuleset      []ValidRule
		expectedErrorCount int
	}{
		{
			name:               "simple plan",
			inputRuleset:       []string{"aws"},
			expectedErrorCount: 0,
		},
		{
			name:               "simple parsing error",
			inputRuleset:       []string{"aws+"},
			expectedErrorCount: 1,
		},
		//TODO cover more cases
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//given
			rulesService := fixRulesService()
			//when
			validRules, validationErrors := rulesService.postParse(&RulesConfig{
				Rules: tc.inputRuleset,
			})
			//then
			if tc.expectedErrorCount == 0 {
				require.NotNil(t, validRules)
				require.Equal(t, 0, len(validationErrors.ParsingErrors))
			} else {
				require.Equal(t, tc.expectedErrorCount, len(validationErrors.ParsingErrors))
				require.Nil(t, validRules)
			}
		})
	}
}

// TODO implement at least the same test cases but starting with fixed ValidRuleset (not using postParse)
func TestValidRuleset_CheckUniqueness(t *testing.T) {

	testCases := []struct {
		name                 string
		ruleset              []string
		duplicateErrorsCount int
	}{
		{name: "simple duplicate",
			ruleset:              []string{"aws", "aws"},
			duplicateErrorsCount: 1,
		},
		{name: "four duplicates",
			ruleset:              []string{"aws", "aws", "aws", "aws"},
			duplicateErrorsCount: 3,
		},
		{name: "simple duplicate with duplicateErrorsCount",
			ruleset:              []string{"aws->EU", "aws->S"},
			duplicateErrorsCount: 1,
		},
		{name: "duplicate with one attribute",
			ruleset:              []string{"aws(PR=x)", "aws(PR=x)"},
			duplicateErrorsCount: 1,
		},
		{name: "no duplicate with one attribute",
			ruleset:              []string{"aws(PR=x)", "aws(PR=y)"},
			duplicateErrorsCount: 0,
		},
		{name: "duplicate with two attributes",
			ruleset:              []string{"aws(PR=x,HR=y)", "aws(PR=x,HR=y)"},
			duplicateErrorsCount: 1,
		},
		{name: "duplicate with two attributes reversed",
			ruleset:              []string{"aws(HR=y,PR=x)", "aws(PR=x,HR=y)"},
			duplicateErrorsCount: 1,
		},
		{name: "no duplicate with two attributes reversed",
			ruleset:              []string{"aws(HR=y,PR=x)", "aws(PR=x,HR=z)"},
			duplicateErrorsCount: 0,
		},
		{name: "duplicate with two attributes reversed and whitespaces",
			ruleset:              []string{"aws ( HR= y,PR=x)", "aws(	PR =x,HR = y )"},
			duplicateErrorsCount: 1,
		},
		{name: "more duplicates with two attributes reversed and whitespaces",
			ruleset:              []string{"aws ( HR= y,PR=x)", "aws(	PR =x,HR = y )", "azure ( HR= a,PR=b)", "azure(	PR =b,HR = a )"},
			duplicateErrorsCount: 2,
		},
		{name: "not duplicate",
			ruleset:              []string{"aws", "azure"},
			duplicateErrorsCount: 0,
		},
		{name: "duplicate amongst many",
			ruleset:              []string{"aws", "azure", "aws"},
			duplicateErrorsCount: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//given
			rulesService := fixRulesService()
			validRules, _ := rulesService.postParse(&RulesConfig{
				Rules: tc.ruleset,
			})
			//when
			ok, duplicateErrors := validRules.checkUniqueness()
			//then
			assert.Equal(t, tc.duplicateErrorsCount, len(duplicateErrors))
			assert.Equal(t, len(duplicateErrors) == 0, ok)
		})
	}
}

func fixRulesService() *RulesService {

	enabledPlans := append(broker.EnablePlans{}, "aws")
	enabledPlans = append(enabledPlans, "azure")

	rs := &RulesService{
		parser: &SimpleParser{
			enabledPlans: &enabledPlans,
		},
	}

	return rs
}
