package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixedLess(t *testing.T) {
	testCases := []struct {
		name   string
		x, y   *ParsingResult
		output int
	}{
		{
			name:   "plan name lexicographically",
			x:      fixParsingResult("aws", "", "", 0, 0),
			y:      fixParsingResult("azure", "", "", 0, 0),
			output: -1,
		},
		{
			name:   "plan name lexicographically - the same plans no differences",
			x:      fixParsingResult("aws", "", "", 0, 0),
			y:      fixParsingResult("aws", "", "", 0, 0),
			output: 0,
		},
		{
			name:   "number of parsing errors",
			x:      fixParsingResult("aws", "", "", 1, 0),
			y:      fixParsingResult("aws", "", "", 2, 0),
			output: -1,
		},
		{
			name:   "number of processing errors",
			x:      fixParsingResult("aws", "", "", 1, 1),
			y:      fixParsingResult("aws", "", "", 1, 2),
			output: 0,
		},
		{
			name:   "number of processing errors with no parsing errors",
			x:      fixParsingResult("aws", "", "", 0, 1),
			y:      fixParsingResult("aws", "", "", 0, 2),
			output: -1,
		},
		{
			name:   "parsing errors are less than ok",
			x:      fixParsingResult("aws", "", "", 0, 0),
			y:      fixParsingResult("aws", "", "", 1, 0),
			output: 1,
		},
		{
			name:   "processing errors are less than ok",
			x:      fixParsingResult("aws", "", "", 0, 0),
			y:      fixParsingResult("aws", "", "", 0, 1),
			output: 1,
		},
		{
			name:   "one input parameter is less than two",
			x:      fixParsingResult("aws", "<pr>", "", 0, 0),
			y:      fixParsingResult("aws", "<pr>", "<hr>", 0, 0),
			output: -1,
		},
		{
			name:   "without input parameter is less than one",
			x:      fixParsingResult("aws", "", "", 0, 0),
			y:      fixParsingResult("aws", "<pr>", "", 0, 0),
			output: -1,
		},
		{
			name:   "without input parameter is less than the one with one but errors have higher priority and are less than ok",
			x:      fixParsingResult("aws", "<pr>", "", 1, 0),
			y:      fixParsingResult("aws", "", "", 0, 0),
			output: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := lessParsingResult(tc.x, tc.y)
			assert.Equal(t, out, tc.output)
		})
	}
}

func fixParsingResult(plan string, platformRegion, hyperscalerRegion string, parsingErrors int, processingErrors int) *ParsingResult {
	rule := NewRule()
	rule.Plan = plan
	rule.PlatformRegion = platformRegion
	rule.HyperscalerRegion = hyperscalerRegion

	pr := NewParsingResult(plan, rule)
	for i := 0; i < parsingErrors; i++ {
		pr.AddParsingError(nil)
	}
	for i := 0; i < processingErrors; i++ {
		pr.AddProcessingError(nil)
	}

	return pr
}
