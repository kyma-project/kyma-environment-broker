package rules

import "testing"

func TestParsingResults_CheckUniqueness(t *testing.T) {

}

func fixParsingResults() *ParsingResults {
	testParsingResults := NewParsingResults()
	testParsingResults.Apply("aws", &Rule{}, nil)

	return testParsingResults
}
