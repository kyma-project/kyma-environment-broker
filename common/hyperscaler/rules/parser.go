package rules

type Parser interface {
    Parse(ruleEntry string) (*Rule, error)
}

type ParsingResult2 struct {
	OriginalRule string

	Rule *Rule
	// array with errors that occurred during parsing of rule entry
	ParsingErrors []error
	
	// array with errors that occurred after successful rule parsing
	ProcessingErrors []error

	Matched bool

	FinalMatch bool
}

func NewParsingResult2(originalRule string, rule *Rule) *ParsingResult2 {
	return &ParsingResult2{
		OriginalRule: originalRule,
		ParsingErrors: make([]error, 0),
		ProcessingErrors: make([]error, 0),
		Matched: false,
		FinalMatch: false,
		Rule: rule,
	}
}


func (r *ParsingResult2) HasParsingErrors() bool {
	return len(r.ParsingErrors) > 0
}

func (r *ParsingResult2) HasProcessingErrors() bool {
	return len(r.ProcessingErrors) > 0
}

func (r *ParsingResult2) HasErrors() bool {
	return r.HasParsingErrors() || r.HasProcessingErrors()
}

func (r *ParsingResult2) AddProcessingError(err error) {
	r.ProcessingErrors = append(r.ProcessingErrors, err)
}

func (r *ParsingResult2) AddParsingError(err error) {
	r.ParsingErrors = append(r.ParsingErrors, err)
}