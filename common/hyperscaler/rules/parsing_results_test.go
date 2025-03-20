package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsingResults_CheckUniqueness(t *testing.T) {

	testCases := []struct {
		name              string
		ruleset           []string
		invalidRulesCount int
	}{
		{name: "simple duplicate",
			ruleset:           []string{"aws", "aws"},
			invalidRulesCount: 1,
		},
		{name: "four duplicates",
			ruleset:           []string{"aws", "aws", "aws", "aws"},
			invalidRulesCount: 3,
		},
		{name: "simple duplicate with ambiguityErrorCount",
			ruleset:           []string{"aws->EU", "aws->S"},
			invalidRulesCount: 1,
		},
		{name: "duplicate with one attribute",
			ruleset:           []string{"aws(PR=x)", "aws(PR=x)"},
			invalidRulesCount: 1,
		},
		{name: "no duplicate with one attribute",
			ruleset:           []string{"aws(PR=x)", "aws(PR=y)"},
			invalidRulesCount: 0,
		},
		{name: "duplicate with two attributes",
			ruleset:           []string{"aws(PR=x,HR=y)", "aws(PR=x,HR=y)"},
			invalidRulesCount: 1,
		},
		{name: "duplicate with two attributes reversed",
			ruleset:           []string{"aws(HR=y,PR=x)", "aws(PR=x,HR=y)"},
			invalidRulesCount: 1,
		},
		{name: "no duplicate with two attributes reversed",
			ruleset:           []string{"aws(HR=y,PR=x)", "aws(PR=x,HR=z)"},
			invalidRulesCount: 0,
		},
		{name: "duplicate with two attributes reversed and whitespaces",
			ruleset:           []string{"aws ( HR= y,PR=x)", "aws(	PR =x,HR = y )"},
			invalidRulesCount: 1,
		},
		{name: "more duplicates with two attributes reversed and whitespaces",
			ruleset:           []string{"aws ( HR= y,PR=x)", "aws(	PR =x,HR = y )", "azure ( HR= a,PR=b)", "azure(	PR =b,HR = a )"},
			invalidRulesCount: 2,
		},
		{name: "not duplicate",
			ruleset:           []string{"aws", "azure"},
			invalidRulesCount: 0,
		},
		{name: "duplicate amongst many",
			ruleset:           []string{"aws", "azure", "aws"},
			invalidRulesCount: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//given
			rs := fixRulesService()
			parsingResults := rs.parseRuleset(&RulesConfig{
				Rules: tc.ruleset,
			})
			//when
			parsingResults.CheckUniqueness()
			//then
			assert.Equal(t, tc.invalidRulesCount, countRulesWithProcessingErrors(parsingResults.Results))
		})
	}
}
