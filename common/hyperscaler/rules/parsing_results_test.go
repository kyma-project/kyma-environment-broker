package rules

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
)

func TestParsingResults_CheckUniqueness(t *testing.T) {

	testCases := []struct {
		name    string
		ruleset []string
		output  int
	}{
		{name: "simple duplicate",
			ruleset: []string{"aws", "aws"},
			output:  1,
		},
		{name: "not duplicate",
			ruleset: []string{"aws", "azure"},
			output:  0,
		},
		{name: "duplicate amongst many",
			ruleset: []string{"aws", "azure", "aws"},
			output:  1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsingResults := fixParsingResults(tc.ruleset)
			parsingResults.CheckUniqueness()
			assert.Equal(t, tc.output, getProcessingErrorCount(parsingResults.Results))

		})
	}
}

func fixParsingResults(rules []string) *ParsingResults {

	enabledPlans := append(broker.EnablePlans{}, "aws")
	enabledPlans = append(enabledPlans, "azure")

	rulesConfig := &RulesConfig{
		Rules: rules,
	}

	rs := &RulesService{
		parser: &SimpleParser{
			enabledPlans: &enabledPlans,
		},
	}

	return rs.parseRuleset(rulesConfig)
}
