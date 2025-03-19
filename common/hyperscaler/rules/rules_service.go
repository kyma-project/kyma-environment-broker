package rules

import (
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type RulesService struct {
	parser        Parser
	ParsedRuleset *ParsingResults
}

func NewRulesServiceFromFile(rulesFilePath string, enabledPlans *broker.EnablePlans) (*RulesService, error) {

	if rulesFilePath == "" {
		return nil, fmt.Errorf("No HAP rules file path provided")
	}

	log.Printf("Parsing rules from file: %s\n", rulesFilePath)
	file, err := os.Open(rulesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", err)
	}

	rs, err := NewRulesService(file, enabledPlans)
	return rs, err
}

func NewRulesService(file *os.File, enabledPlans *broker.EnablePlans) (*RulesService, error) {
	rulesConfig := &RulesConfig{}

	if file == nil {
		return nil, fmt.Errorf("No HAP rules file provided")
	}

	err := rulesConfig.LoadFromFile(file)
	if err != nil {
		return nil, err
	}

	rs := &RulesService{
		parser: &SimpleParser{
			enabledPlans: enabledPlans,
		},
	}

	rs.ParsedRuleset = rs.parse(rulesConfig)
	return rs, nil
}

func NewRulesServiceFromSlice(rules []string, enabledPlans *broker.EnablePlans) (*RulesService, error) {

	rulesConfig := &RulesConfig{
		Rules: rules,
	}

	rs := &RulesService{
		parser: &SimpleParser{
			enabledPlans: enabledPlans,
		},
	}

	rs.ParsedRuleset = rs.parse(rulesConfig)
	return rs, nil
}

func (rs *RulesService) parse(rulesConfig *RulesConfig) *ParsingResults {

	results := rs.parseRuleset(rulesConfig)

	results.CheckUniqueness()

	results.CheckSignatures()

	results.Results = SortRuleEntries(results.Results)

	return results
}

func (rs *RulesService) parseRuleset(rulesConfig *RulesConfig) *ParsingResults {
	results := NewParsingResults()

	for _, entry := range rulesConfig.Rules {
		parsedRule, err := rs.parser.Parse(entry)

		results.Apply(entry, parsedRule, err)
	}
	return results
}

// MatchProvisioningAttributes finds the matching rule for the given provisioning attributes and provide values needed to create labels, which must be used to find proper secret binding.
func (rs *RulesService) MatchProvisioningAttributes(provisioningAttributes *ProvisioningAttributes) (Result, bool) {
	var result Result
	found := false
	for _, parsingResult := range rs.ParsedRuleset.Results {
		if parsingResult.Rule.Matched(provisioningAttributes) {
			result = parsingResult.Rule.ProvideResult(provisioningAttributes)
			found = true
		}
	}

	return result, found
}

func (rs *RulesService) Match(data *ProvisioningAttributes) map[uuid.UUID]*MatchingResult {
	var matchingResults map[uuid.UUID]*MatchingResult = make(map[uuid.UUID]*MatchingResult)

	var lastMatch *MatchingResult = nil
	for _, result := range rs.ParsedRuleset.Results {
		if !result.HasParsingErrors() {
			matchingResult := &MatchingResult{
				ParsingResultID:        result.ID,
				OriginalRule:           result.OriginalRule,
				Rule:                   *result.Rule,
				ProvisioningAttributes: data,
			}

			matchingResult.Matched = result.Rule.Matched(data)
			if matchingResult.Matched {
				lastMatch = matchingResult
			}

			matchingResults[result.ID] = matchingResult
		}
	}

	if lastMatch != nil {
		lastMatch.FinalMatch = true
	}

	return matchingResults
}

// TODO redesign this method
func (rs *RulesService) FirstParsingError() error {
	for _, result := range rs.ParsedRuleset.Results {
		if result.HasErrors() {
			buffer := ""
			var printer *Printer = NewNoColor(func(format string, a ...interface{}) {
				buffer += fmt.Sprintf(format, a...)
			})

			printer.Print(rs.ParsedRuleset.Results, nil)
			return fmt.Errorf("Parsing errors occurred during rules parsing, results are: %s", buffer)
		}
	}

	return nil
}
