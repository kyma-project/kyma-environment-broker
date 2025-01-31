package rules

type Evaluator struct {
	rules []*Rule
}

func NewEvaluator(parser Parser) *Evaluator {
    return &Evaluator{
        // rules: parser.Parse(),
    }
}

/**
 * Validate rules.
 */
func (e *Evaluator) Validate() bool {
	return true
}

/**
 * Evaluate rules and output search labels.
 */
func (e *Evaluator) Evaluate( /*srk*/ ) string {
	matchedRules := e.findMatchedRules( /*srk*/ )

	// sort rules by priority
	matchedRules = e.sortRules(matchedRules)

	// apply one found rule
    return matchedRules[0].Labels()
}

func (e *Evaluator) sortRules(matchedRules []*Rule) []*Rule {
	panic("unimplemented")
}

func (e *Evaluator) findMatchedRules() []*Rule {
	panic("unimplemented")
}
