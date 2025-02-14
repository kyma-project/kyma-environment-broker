package rules

import "sort"

func SortRuleEntries(entries []ParsingResult) []ParsingResult {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Err != nil || entries[j].Err != nil {
			return true
		}

		if entries[i].Rule.Plan != entries[j].Rule.Plan {
			return entries[i].Rule.Plan < entries[j].Rule.Plan
		}

		return entries[i].Rule.NumberOfInputAttributes() < entries[j].Rule.NumberOfInputAttributes()
	})

	return entries
}
