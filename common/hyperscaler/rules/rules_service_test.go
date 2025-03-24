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
		{name: "simple duplicate with ambiguityErrorCount",
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

func TestValidRuleset_CheckAmbiguity(t *testing.T) {

	testCases := []struct {
		name                string
		ruleset             []string
		ambiguityErrorCount int
	}{
		{name: "simple plan",
			ruleset:             []string{"aws"},
			ambiguityErrorCount: 0,
		},
		{name: "basic ambiguity",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)"},
			ambiguityErrorCount: 1,
		},
		{name: "basic ambiguity - but disambiguation added",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "aws(PR=x,HR=y)"},
			ambiguityErrorCount: 0,
		},
		{name: "basic ambiguity - wrong disambiguation added",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "azure(PR=x,HR=y)"},
			ambiguityErrorCount: 1,
		},
		{name: "basic ambiguity - wrong disambiguation added",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "aws(PR=x,HR=z)"},
			ambiguityErrorCount: 1,
		},
		{name: "this is not basic ambiguity",
			ruleset:             []string{"aws(PR=x)", "azure(HR=y)"},
			ambiguityErrorCount: 0,
		},
		{name: "double ambiguity",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "aws(HR=z)"},
			ambiguityErrorCount: 2,
		},
		{name: "double ambiguity - insufficient disambiguation",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)"},
			ambiguityErrorCount: 1,
		},
		{name: "double ambiguity - sufficient disambiguation",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)", "aws(PR=x,HR=z)"},
			ambiguityErrorCount: 0,
		},
		{name: "double ambiguity - wrong disambiguation",
			ruleset:             []string{"aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)", "azure(PR=x,HR=z)"},
			ambiguityErrorCount: 1,
		},
		{name: "quadruple ambiguity",
			ruleset:             []string{"aws(PR=v)", "aws(PR=x)", "aws(HR=y)", "aws(HR=z)"},
			ambiguityErrorCount: 4,
		},
		{name: "double ambiguity - insufficient disambiguation - missing 3",
			ruleset:             []string{"aws(PR=v)", "aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)"},
			ambiguityErrorCount: 3,
		},
		{name: "double ambiguity - insufficient disambiguation - missing 2",
			ruleset:             []string{"aws(PR=v)", "aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)", "aws(PR=x,HR=z)"},
			ambiguityErrorCount: 2,
		},
		{name: "double ambiguity - insufficient disambiguation - missing 1",
			ruleset:             []string{"aws(PR=v)", "aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)", "aws(PR=x,HR=z)", "aws(PR=v,HR=z)"},
			ambiguityErrorCount: 1,
		},
		{name: "double ambiguity - sufficient disambiguation",
			ruleset:             []string{"aws(PR=v)", "aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)", "aws(PR=x,HR=z)", "aws(PR=v,HR=z)", "aws(PR=v,HR=y)"},
			ambiguityErrorCount: 0,
		},
		{name: "double ambiguity - wrong disambiguation",
			ruleset:             []string{"aws(PR=v)", "aws(PR=x)", "aws(HR=y)", "aws(HR=z)", "aws(PR=x,HR=y)", "azure(PR=x,HR=z)", "aws(PR=v,HR=z)", "aws(PR=v,HR=y)"},
			ambiguityErrorCount: 1,
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
			ok, ambiguityErrors := validRules.checkUnambiguity()
			//then
			assert.Equal(t, tc.ambiguityErrorCount, len(ambiguityErrors))
			assert.Equal(t, len(ambiguityErrors) == 0, ok)
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
