package rules

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type RulesService struct {
	sort      bool
	unique    bool
	signature bool

	parser Parser
	Parsed *ParsingResults
}

func NewRulesServiceFromFile(rulesFilePath string, enabledPlans *broker.EnablePlans, sort, unique, signature bool) (*RulesService, error) {
	rulesConfig := &RulesConfig{}

	if rulesFilePath == "" {
		return nil, fmt.Errorf("No HAP rules file path provided")
	}

	log.Printf("Parsing rules from file: %s\n", rulesFilePath)
	err := rulesConfig.Load(rulesFilePath)
	if err != nil {
		return nil, err
	}

	rs := &RulesService{
		parser: &SimpleParser{
			enabledPlans: enabledPlans,
		},
		sort:      sort,
		unique:    unique,
		signature: signature,
	}

	rs.Parsed = rs.parse(rulesConfig)
	return rs, err
}

func NewRulesServiceFromString(rules string, enabledPlans *broker.EnablePlans, sort, unique, signature bool) (*RulesService, error) {
	entries := strings.Split(rules, ";")

	rulesConfig := &RulesConfig{
		Rules: entries,
	}

	rs := &RulesService{
		parser: &SimpleParser{
			enabledPlans: enabledPlans,
		},
		sort:      sort,
		unique:    unique,
		signature: signature,
	}

	rs.Parsed = rs.parse(rulesConfig)
	return rs, nil
}

func (rs *RulesService) parse(rulesConfig *RulesConfig) *ParsingResults {
	results := NewParsingResults()

	for _, entry := range rulesConfig.Rules {
		rule, err := rs.parser.Parse(entry)

		results.Apply(entry, rule, err)
	}

	if rs.sort {
		results.Results = SortRuleEntries(results.Results)
	}

	if rs.unique {
		results.CheckUniqueness()
	}

	if rs.signature {
		results.CheckSignatures()
	}

	if rs.sort {
		results.Results = SortRuleEntries(results.Results)
	}

	return results
}

func (rs *RulesService) Match(data *ProvisioningAttributes) map[uuid.UUID]*MatchingResult {
	var matchingResults map[uuid.UUID]*MatchingResult = make(map[uuid.UUID]*MatchingResult)

	var lastMatch *MatchingResult = nil
	for _, result := range rs.Parsed.Results {
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

func (rs *RulesService) FailOnParsingErrors() error {
	for _, result := range rs.Parsed.Results {
		if result.HasErrors() {
			buffer := ""
			var printer *Printer = NewNoColor(func(format string, a ...interface{}) {
				buffer += fmt.Sprintf(format, a...)
			})

			printer.Print(rs.Parsed.Results, nil)
			log.Fatalf("Parsing errors occurred during rules parsing")
			return fmt.Errorf("Parsing errors occurred during rules parsing, results are: %s", buffer)
		}
	}

	return nil
}
