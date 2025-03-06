package rules

import "github.com/google/uuid"

type ParsingResult struct {
	ID uuid.UUID

	OriginalRule string

	Rule *Rule
	// array with errors that occurred during parsing of rule entry
	ParsingErrors []error

	// array with errors that occurred after successful rule parsing
	ProcessingErrors []error
}

func NewParsingResult(originalRule string, rule *Rule) *ParsingResult {
	return &ParsingResult{
		ID:               uuid.New(),
		OriginalRule:     originalRule,
		ParsingErrors:    make([]error, 0),
		ProcessingErrors: make([]error, 0),
		Rule:             rule,
	}
}

func (r *ParsingResult) HasParsingErrors() bool {
	return len(r.ParsingErrors) > 0
}

func (r *ParsingResult) HasProcessingErrors() bool {
	return len(r.ProcessingErrors) > 0
}

func (r *ParsingResult) HasErrors() bool {
	return r.HasParsingErrors() || r.HasProcessingErrors()
}

func (r *ParsingResult) AddProcessingError(err error) {
	r.ProcessingErrors = append(r.ProcessingErrors, err)
}

func (r *ParsingResult) AddParsingError(err error) {
	r.ParsingErrors = append(r.ParsingErrors, err)
}
