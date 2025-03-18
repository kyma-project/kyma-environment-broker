package rules

import "fmt"

type ParsingResults struct {
	resolvedRules map[string]*ParsingResult
	uniqueResults []*ParsingResult

	Results []*ParsingResult
}

func (p *ParsingResults) Apply(entry string, rule *Rule, err error) {
	result := NewParsingResult(entry, rule)

	if err != nil {
		result.AddParsingError(err)
	} else {
		if rule.IsResolved() {
			p.resolvedRules[rule.StringNoLabels()] = result
		}
	}

	p.Results = append(p.Results, result)
}

func NewParsingResults() *ParsingResults {
	return &ParsingResults{
		Results:       make([]*ParsingResult, 0),
		resolvedRules: make(map[string]*ParsingResult),
		uniqueResults: make([]*ParsingResult, 0),
	}
}

func (p *ParsingResults) IsResolved(resolvingKey string) bool {
	_, resolvingRuleExists := p.resolvedRules[resolvingKey]
	return resolvingRuleExists
}

func (p *ParsingResults) CheckUniqueness() {

	uniquenessSet := make(map[string]*ParsingResult)
	resultsWithoutErrors := getResultsWithoutErrors(p.Results)

	for _, result := range resultsWithoutErrors {

		key := result.Rule.SignatureWithValues()

		item, alreadyExists := uniquenessSet[key]

		if alreadyExists {
			result.AddProcessingError(fmt.Errorf("Duplicated rule with previously defined rule: '%s'", item.Rule.StringNoLabels()))
		} else {
			uniquenessSet[key] = result
		}
	}
}

func (p *ParsingResults) CheckSignatures() {
	uniqueResults := make([]*ParsingResult, 0, len(p.Results))
	proposedResolutions := make(map[string]bool)
	signatureSet := NewSignatureSet(p.Results)

	for _, result := range p.Results {

		if result.HasErrors() {
			uniqueResults = append(uniqueResults, result)
			continue
		}

		var mirroredSignatureError error = nil

		mirrored := signatureSet.Mirrored(result.Rule)

		for _, mirroredSignatureItem := range mirrored {

			possibleResolvingRule := result.Rule.Combine(*mirroredSignatureItem.Rule)

			resolvingKey := possibleResolvingRule.StringNoLabels()

			_, alreadyProposed := proposedResolutions[resolvingKey]

			if !p.IsResolved(resolvingKey) && !alreadyProposed {
				mirroredSignatureError = fmt.Errorf("Ambiguous with previously defined entry: '%s', consider introducing a resolving entry '%s'", mirroredSignatureItem.Rule.StringNoLabels(), resolvingKey)

				proposedResolutions[resolvingKey] = true

				result.AddProcessingError(mirroredSignatureError)
			}
		}

		uniqueResults = append(uniqueResults, result)
	}

	p.Results = uniqueResults
}

func countRulesWithProcessingErrors(results []*ParsingResult) int {
	errorCount := 0

	for _, result := range results {
		if result.HasProcessingErrors() {
			errorCount++
		}
	}

	return errorCount
}

func getResultsWithoutErrors(results []*ParsingResult) []*ParsingResult {
	resultsWithoutErrors := make([]*ParsingResult, 0, len(results))

	for _, result := range results {
		if !result.HasErrors() {
			resultsWithoutErrors = append(resultsWithoutErrors, result)
		}
	}

	return resultsWithoutErrors
}
