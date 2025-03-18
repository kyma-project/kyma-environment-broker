package rules

import (
	"cmp"

	"golang.org/x/exp/slices"
)

var lessParsingResult = func(x, y *ParsingResult) int {
	// less errors precede more errors
	if len(x.ParsingErrors) != 0 && len(y.ParsingErrors) != 0 {
		return len(x.ParsingErrors) - len(y.ParsingErrors)
	}

	// Errors precede OK
	if len(x.ParsingErrors) != 0 {
		return -1
	}

	if len(y.ParsingErrors) != 0 {
		return 1
	}

	// less errors precede more errors
	if len(x.ProcessingErrors) != 0 && len(y.ProcessingErrors) != 0 {
		return len(x.ProcessingErrors) - len(y.ProcessingErrors)
	}

	// Errors precede OK
	if len(x.ProcessingErrors) != 0 {
		return -1
	}

	if len(y.ProcessingErrors) != 0 {
		return 1
	}

	// plans are sorted lexicographically
	if x.Rule.Plan != y.Rule.Plan {
		return cmp.Compare(x.Rule.Plan, y.Rule.Plan)
	}

	// less input attributes precede more input attributes
	return x.Rule.NumberOfNonEmptyInputAttributes() - y.Rule.NumberOfNonEmptyInputAttributes()
}

func SortRuleEntries(entries []*ParsingResult) []*ParsingResult {
	slices.SortStableFunc(entries, lessParsingResult)
	return entries
}
