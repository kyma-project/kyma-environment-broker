package rules

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Evaluator struct {
	rules map[string][]*Rule
}

func NewEvaluator(rules []*Rule) *Evaluator {
    evaluator := &Evaluator{
        rules: make(map[string][]*Rule),
    }

	for _, rule := range rules {
		if _, exists := evaluator.rules[rule.Plan]; !exists {
			evaluator.rules[rule.Plan] = make([]*Rule, 0)
		}
		evaluator.rules[rule.Plan] = append(evaluator.rules[rule.Plan], rule)	
	}

	return evaluator
}

/**
 * Evaluate rules and output search labels.
 */
func (e *Evaluator) Evaluate(matchableAttributes *MatchableAttributes) ([]*Rule, error) {

	if _, ok := broker.PlanIDsMapping[matchableAttributes.Plan]; !ok {
		return nil, fmt.Errorf("invalid plan %s passed as input to matching process", matchableAttributes.Plan)
	}

	matchedRules := make([]*Rule, 0)
	for _, rule := range e.rules[matchableAttributes.Plan] {
		if rule.Matched(matchableAttributes) {
			matchedRules = append(matchedRules, rule)
		}
	}

	// apply one found rule
    return matchedRules, nil
}
