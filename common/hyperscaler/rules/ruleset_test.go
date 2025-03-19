package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidRule_KeyString(t *testing.T) {
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
