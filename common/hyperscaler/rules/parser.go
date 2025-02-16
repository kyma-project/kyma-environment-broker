package rules

type Parser interface {
    Parse(ruleEntry string) (*Rule, error)
}

type ParsingResult struct {
	OriginalRule string
	Rule *Rule
	Err  error
	Matched bool
	FinalMatch bool
}