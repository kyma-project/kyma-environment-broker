package rules

import (
	"github.com/google/uuid"
)

type Printf func(format string, a ...interface{})

type Printer struct {
	colorError   string
	colorOk      string
	colorNeutral string
	colorMatched string
	print        Printf
}

func NewColored(print Printf) *Printer {
	return &Printer{
		colorError:   "\033[0;31m",
		colorOk:      "\033[32m",
		colorNeutral: "\033[0m",
		colorMatched: "\033[34m",
		print:        print,
	}
}

func NewNoColor(print Printf) *Printer {
	return &Printer{
		colorError:   "",
		colorOk:      "",
		colorNeutral: "",
		colorMatched: "",
		print:        print,
	}
}

func (p *Printer) Print(results []*ParsingResult, matchingResults map[uuid.UUID]*MatchingResult) {
	for _, result := range results {
		p.print("-> ")
		hasErrors := result.HasErrors()
		if hasErrors {
			p.print("%s Error %s", p.colorError, p.colorNeutral)
		} else {
			p.print("%s %5s %s", p.colorOk, "OK", p.colorNeutral)
		}

		if result.Rule != nil && !hasErrors {
			p.print(" %s", result.Rule.String())
		}

		if hasErrors {
			p.print(" %s", result.OriginalRule)
			for _, err := range result.ParsingErrors {
				p.print("\n - %s", err)
			}

			for _, err := range result.ProcessingErrors {
				p.print("\n - %s", err)
			}

		}

		if !hasErrors && matchingResults != nil {

			matchingResult := matchingResults[result.ID]

			if matchingResult.Matched && !matchingResult.FinalMatch {
				p.print("%s Matched %s ", p.colorMatched, p.colorNeutral)
			} else if matchingResult.FinalMatch {
				p.print("%s Matched, Selected %s ", p.colorMatched, p.colorNeutral)
			}
		}

		p.print("\n")
	}
}
