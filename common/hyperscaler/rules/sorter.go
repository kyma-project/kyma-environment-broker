package rules

import (
	"sort"
)

func SortRuleEntries(entries []*ParsingResult) []*ParsingResult {
	sort.SliceStable(entries, func(i, j int) bool {
		
		if len(entries[i].ParsingErrors) != 0 && len(entries[j].ParsingErrors) != 0 {
			return len(entries[i].ParsingErrors) < len(entries[j].ParsingErrors)
		}

		if len(entries[i].ParsingErrors) != 0 || len(entries[j].ParsingErrors) != 0  {
			return true
		}

		if len(entries[i].ParsingErrors) != 0 && len(entries[j].ParsingErrors) != 0 {
			return len(entries[i].ProcessingErrors) < len(entries[j].ProcessingErrors)
		}


		if len(entries[i].ProcessingErrors) != 0 || len(entries[j].ProcessingErrors) != 0  {
			return true
		}

		if entries[i].Rule.Plan != entries[j].Rule.Plan {
			return entries[i].Rule.Plan < entries[j].Rule.Plan
		}

		return entries[i].Rule.NumberOfInputAttributes() < entries[j].Rule.NumberOfInputAttributes()
	})

	return entries
}
