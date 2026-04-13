package rules

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRulesServiceFromFile(t *testing.T) {

	t.Run("should return error when file path is empty", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("", sets.New[string](), sets.New[string]())

		// then
		require.Error(t, err)
		require.Nil(t, service)
		require.Equal(t, "No HAP rules file path provided", err.Error())
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// when
		service, err := NewRulesServiceFromFile("nonexistent.yaml", sets.New[string](), sets.New[string]())

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

	t.Run("should return error when YAML file is corrupted", func(t *testing.T) {
		// given
		content := "corrupted_content"

		tmpfile, err := CreateTempFile(content)
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		// when
		service, err := NewRulesServiceFromFile(tmpfile, sets.New[string](), sets.New[string]())

		// then
		require.Error(t, err)
		require.Nil(t, service)
	})

}

func TestNewRulesService(t *testing.T) {

	t.Run("returns error for nil file", func(t *testing.T) {
		rs, err := NewRulesService(nil, sets.New[string](), sets.New[string]())

		require.Error(t, err)
		assert.Nil(t, rs)
		assert.Equal(t, "No HAP rules file provided", err.Error())
	})

	t.Run("returns valid service and no error for correct rules", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n  - aws\n  - azure\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New("aws", "azure"), sets.New("aws", "azure"))

		require.NoError(t, err)
		require.NotNil(t, rs)
		assert.True(t, rs.IsRulesetValid())
		assert.NotNil(t, rs.ValidRules)
		assert.Len(t, rs.ValidRules.Rules, 2)
	})

	t.Run("returns service with error and correct prefix for empty rules list", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New[string](), sets.New[string]())

		require.Error(t, err)
		require.NotNil(t, rs)
		assert.False(t, rs.IsRulesetValid())
		assert.Contains(t, err.Error(), "There are errors in subscription secret rules configuration:")
	})

	t.Run("returns service with error for parsing errors and error message contains all messages", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n  - aws(\n  - azure(\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New("aws", "azure"), sets.New("aws", "azure"))

		require.Error(t, err)
		require.NotNil(t, rs)
		assert.False(t, rs.IsRulesetValid())
		assert.Contains(t, err.Error(), "There are errors in subscription secret rules configuration:")
		require.NotNil(t, rs.ValidationInfo)
		assert.Len(t, rs.ValidationInfo.ParsingErrors, 2)
		for _, parsingErr := range rs.ValidationInfo.ParsingErrors {
			assert.Contains(t, err.Error(), parsingErr.Error())
		}
	})

	t.Run("returns service with error for duplicate rules and error message contains all messages", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n  - aws\n  - aws\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New("aws"), sets.New("aws"))

		require.Error(t, err)
		require.NotNil(t, rs)
		assert.False(t, rs.IsRulesetValid())
		assert.Contains(t, err.Error(), "There are errors in subscription secret rules configuration:")
		require.NotNil(t, rs.ValidationInfo)
		assert.Len(t, rs.ValidationInfo.DuplicateErrors, 1)
		assert.Contains(t, err.Error(), rs.ValidationInfo.DuplicateErrors[0].Error())
	})

	t.Run("returns service with error for ambiguous rules and error message contains all messages", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n  - aws(PR=x)\n  - aws(HR=y)\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New("aws"), sets.New("aws"))

		require.Error(t, err)
		require.NotNil(t, rs)
		assert.False(t, rs.IsRulesetValid())
		assert.Contains(t, err.Error(), "There are errors in subscription secret rules configuration:")
		require.NotNil(t, rs.ValidationInfo)
		assert.Len(t, rs.ValidationInfo.AmbiguityErrors, 1)
		assert.Contains(t, err.Error(), rs.ValidationInfo.AmbiguityErrors[0].Error())
	})

	t.Run("returns service with error for plan errors and error message contains all messages", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n  - aws\n  - unknown-plan\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New("aws"), sets.New("aws"))

		require.Error(t, err)
		require.NotNil(t, rs)
		assert.False(t, rs.IsRulesetValid())
		assert.Contains(t, err.Error(), "There are errors in subscription secret rules configuration:")
		require.NotNil(t, rs.ValidationInfo)
		assert.Len(t, rs.ValidationInfo.PlanErrors, 1)
		assert.Contains(t, err.Error(), rs.ValidationInfo.PlanErrors[0].Error())
	})

	t.Run("error message contains all individual error messages joined", func(t *testing.T) {
		tmpfile, err := CreateTempFile("rule:\n  - aws(\n  - azure(\n  - gcp(\n")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpfile) }()

		file, err := os.Open(tmpfile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		rs, err := NewRulesService(file, sets.New("aws", "azure", "gcp"), sets.New[string]())

		require.Error(t, err)
		require.NotNil(t, rs)
		assert.Contains(t, err.Error(), "There are errors in subscription secret rules configuration:")
		require.NotNil(t, rs.ValidationInfo)
		for _, e := range rs.ValidationInfo.All() {
			assert.Contains(t, err.Error(), e.Error())
		}
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
			inputRuleset:       []string{"aws("},
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

func TestRulesService_CheckPlans(t *testing.T) {
	for tn, tc := range map[string]struct {
		ruleset         []string
		allowedPlans    sets.Set[string]
		requiredPlans   sets.Set[string]
		plansErrorCount int
	}{
		"all good": {
			ruleset:         []string{"aws", "azure"},
			allowedPlans:    sets.New("aws", "azure", "gcp"),
			requiredPlans:   sets.New("aws", "azure"),
			plansErrorCount: 0,
		},
		"missing required plan": {
			ruleset:         []string{"aws", "azure"},
			allowedPlans:    sets.New("aws", "azure", "gcp"),
			requiredPlans:   sets.New("aws", "gcp"),
			plansErrorCount: 1,
		},
		"not known plan": {
			ruleset:         []string{"aws", "azure"},
			allowedPlans:    sets.New("aws"),
			requiredPlans:   sets.New("aws"),
			plansErrorCount: 1,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			//given
			rulesService := fixRulesService()
			rulesService.requiredPlans = tc.requiredPlans
			rulesService.allowedPlans = tc.allowedPlans
			validRules, _ := rulesService.postParse(&RulesConfig{
				Rules: tc.ruleset,
			})

			//when
			ok, planErrors := validRules.checkPlans(tc.allowedPlans, tc.requiredPlans)

			//then
			assert.Equal(t, tc.plansErrorCount, len(planErrors))
			assert.Equal(t, len(planErrors) == 0, ok)

		})
	}
}

func fixRulesService() *RulesService {

	rs := &RulesService{
		parser: &SimpleParser{},
	}

	return rs
}
