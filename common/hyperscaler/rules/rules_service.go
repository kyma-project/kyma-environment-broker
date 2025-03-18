package rules

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
)

type RulesService struct {
	parser Parser
	Parsed *ParsingResults
}

func NewRulesServiceFromFile(rulesFilePath string) (*RulesService, error) {

	if rulesFilePath == "" {
		return nil, fmt.Errorf("No HAP rules file path provided")
	}

	log.Printf("Parsing rules from file: %s\n", rulesFilePath)
	file, err := os.Open(rulesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", err)
	}

	rs, err := NewRulesService(file)
	return rs, err
}

func NewRulesService(file *os.File) (*RulesService, error) {
	rulesConfig := &RulesConfig{}

	if file == nil {
		return nil, fmt.Errorf("No HAP rules file provided")
	}

	err := rulesConfig.LoadFromFile(file)
	if err != nil {
		return nil, err
	}

	rs := &RulesService{
		parser: &SimpleParser{},
	}

	rs.Parsed = rs.parse(rulesConfig)
	return rs, err
}

func NewRulesServiceFromString(rules string) (*RulesService, error) {
	entries := strings.Split(rules, ";")

	rulesConfig := &RulesConfig{
		Rules: entries,
	}

	rs := &RulesService{
		parser: &SimpleParser{},
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

	results.Results = SortRuleEntries(results.Results)

	results.CheckUniqueness()

	results.CheckSignatures()

	results.Results = SortRuleEntries(results.Results)

	return results
}

// MatchProvisioningAttributes finds the matching rule for the given provisioning attributes and provide values needed to create labels, which must be used to find proper secret binding.
func (rs *RulesService) MatchProvisioningAttributes(provisioningAttributes *ProvisioningAttributes) (Result, bool) {
	var result Result
	found := false
	for _, parsingResult := range rs.Parsed.Results {
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

func (rs *RulesService) FirstParsingError() error {
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

func (rs *RulesService) ParsingErrors() []error {
	var errors []error
	for _, result := range rs.Parsed.Results {
		if result.HasParsingErrors() {
			errors = append(errors, result.ParsingErrors...)
		}
	}

	return errors
}

func (rs *RulesService) ProcessingErrors() []error {
	var errors []error
	for _, result := range rs.Parsed.Results {
		if result.HasProcessingErrors() {
			errors = append(errors, result.ProcessingErrors...)
		}
	}

	return errors
}
