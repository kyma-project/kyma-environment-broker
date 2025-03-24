package rules

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type RulesService struct {
	parser         Parser
	ParsedRuleset  *ParsingResults
	ValidRules     *ValidRuleset
	ValidationInfo *ValidationErrors
}

func NewRulesServiceFromFile(rulesFilePath string, enabledPlans *broker.EnablePlans) (*RulesService, error) {

	if rulesFilePath == "" {
		return nil, fmt.Errorf("No HAP rules file path provided")
	}

	slog.Info(fmt.Sprintf("Parsing rules from file: %s\n", rulesFilePath))
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

	rs.ParsedRuleset = rs.process(rulesConfig)
	rs.ValidRules, rs.ValidationInfo = rs.processAndValidate(rulesConfig)
	return rs, err
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

	rs.ParsedRuleset = rs.process(rulesConfig)
	rs.ValidRules, rs.ValidationInfo = rs.processAndValidate(rulesConfig)
	return rs, nil
}

func (rs *RulesService) processAndValidate(rulesConfig *RulesConfig) (*ValidRuleset, *ValidationErrors) {

	validRuleset, validationErrors := rs.postParse(rulesConfig)
	if len(validationErrors.ParsingErrors) > 0 {
		return nil, validationErrors
	}

	ok, duplicateErrors := validRuleset.checkUniqueness()
	if !ok {
		validationErrors.DuplicateErrors = append(validationErrors.DuplicateErrors, duplicateErrors...)
		return nil, validationErrors
	}

	ok, ambiguityErrors := validRuleset.checkUnambiguity()
	if !ok {
		validationErrors.AmbiguityErrors = append(validationErrors.AmbiguityErrors, ambiguityErrors...)
		return nil, validationErrors
	}
	return validRuleset, nil
}

func (rs *RulesService) postParse(rulesConfig *RulesConfig) (*ValidRuleset, *ValidationErrors) {
	validRuleset := NewValidRuleset()
	validationErrors := NewValidationErrors()

	for ruleNo, rawRule := range rulesConfig.Rules {
		rule, err := rs.parser.Parse(rawRule)
		if err != nil {
			validationErrors.ParsingErrors = append(validationErrors.ParsingErrors, err)
		} else {
			validRule := toValidRule(rule, rawRule, ruleNo)
			validRuleset.Rules = append(validRuleset.Rules, *validRule)
		}
	}

	if len(validationErrors.ParsingErrors) > 0 {
		return nil, validationErrors
	}

	return validRuleset, validationErrors
}

func (rs *RulesService) process(rulesConfig *RulesConfig) *ParsingResults {
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

func (rs *RulesService) getSortedRulesForPlan(plan string) []ValidRule {
	rulesForPlan := make([]ValidRule, 0)
	for _, validRule := range rs.ValidRules.Rules {
		if validRule.Plan.literal == plan {
			rulesForPlan = append(rulesForPlan, validRule)
		}
	}
	//sort rules by MatchAnyCount
	slices.SortStableFunc(rulesForPlan, func(x, y ValidRule) int {
		return x.MatchAnyCount - y.MatchAnyCount
	})
	return rulesForPlan
}

func (rs *RulesService) MatchProvisioningAttributesWithValidRuleset(provisioningAttributes *ProvisioningAttributes) (Result, bool) {
	if rs.ValidRules == nil || len(rs.ValidRules.Rules) == 0 {
		slog.Warn("No valid ruleset or empty valid ruleset")
		return Result{}, false
	}

	rulesForPlan := rs.getSortedRulesForPlan(provisioningAttributes.Plan)

	if len(rulesForPlan) == 0 {
		slog.Warn(fmt.Sprintf("No valid rules for plan: %s", provisioningAttributes.Plan))
		return Result{}, false
	}

	//find first matching rule which is the most specific one (lowest MatchAnyCount)
	var result Result
	found := false
	for _, validRule := range rulesForPlan {
		//plan is already matched
		if validRule.matchInputParameters(provisioningAttributes) {
			result = validRule.toResult(provisioningAttributes)
			found = true
			break
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

func (rs *RulesService) FirstParsingError() error {
	for _, result := range rs.ParsedRuleset.Results {
		if result.HasErrors() {
			buffer := ""
			var printer *Printer = NewNoColor(func(format string, a ...interface{}) {
				buffer += fmt.Sprintf(format, a...)
			})

			printer.Print(rs.ParsedRuleset.Results, nil)
			return fmt.Errorf("parsing errors occurred during rules parsing, results are: %s", buffer)
		}
	}

	return nil
}

func toValidRule(rule *Rule, rawRule string, ruleNo int) *ValidRule {
	vr := &ValidRule{
		Plan: PatternAttribute{
			literal: rule.Plan,
		},
		PlatformRegion: PatternAttribute{
			literal: rule.PlatformRegion,
		},
		HyperscalerRegion: PatternAttribute{
			literal: rule.HyperscalerRegion,
		},
		Shared:                  rule.Shared,
		EuAccess:                rule.EuAccess,
		PlatformRegionSuffix:    rule.PlatformRegionSuffix,
		HyperscalerRegionSuffix: rule.HyperscalerRegionSuffix,
	}
	if vr.PlatformRegion.literal == "" {
		vr.PlatformRegion.matchAny = true
		vr.MatchAnyCount++
	}
	if vr.HyperscalerRegion.literal == "" {
		vr.HyperscalerRegion.matchAny = true
		vr.MatchAnyCount++
	}
	vr.RawData = RawData{
		Rule:   rawRule,
		RuleNo: ruleNo,
	}
	return vr
}
