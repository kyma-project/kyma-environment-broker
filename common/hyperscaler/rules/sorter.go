package rules

import "sort"

func SortRuleEntries(entries []ParsingResult) []ParsingResult {
    sort.SliceStable(entries, func(i, j int) bool {
        return entries[i].Err == nil && entries[i].Rule.Plan < entries[j].Rule.Plan
    });

    sort.SliceStable(entries, func(i, j int) bool {
        return entries[i].Err == nil && entries[i].Rule.NumberOfInputAtributes() < entries[j].Rule.NumberOfInputAtributes()
    });

    return entries
}
