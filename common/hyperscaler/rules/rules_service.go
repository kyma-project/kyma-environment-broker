package rules

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"k8s.io/apimachinery/pkg/util/sets"
)

type RulesService struct {
	parser         Parser
	ValidRules     *ValidRuleset
	ValidationInfo *ValidationErrors
	requiredPlans  sets.Set[string]
	allowedPlans   sets.Set[string]
}

func NewRulesServiceFromFile(rulesFilePath string, allowedPlans sets.Set[string], requiredPlans sets.Set[string]) (*RulesService, error) {

	if rulesFilePath == "" {
		return nil, fmt.Errorf("No HAP rules file path provided")
	}

	file, err := os.Open(rulesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", err)
	}

	rs, err := NewRulesService(file, allowedPlans, requiredPlans)
	return rs, err
}

func (rs *RulesService) IsRulesetValid() bool {
	return rs.ValidRules != nil && len(rs.ValidRules.Rules) > 0
}

func NewRulesService(file *os.File, allowedPlans sets.Set[string], requiredPlans sets.Set[string]) (*RulesService, error) {
	rulesConfig := &RulesConfig{}

	if file == nil {
		return nil, fmt.Errorf("No HAP rules file provided")
	}

	err := rulesConfig.LoadFromFile(file)
	if err != nil {
		return nil, err
	}

	rs := &RulesService{
		parser:        &SimpleParser{},
		requiredPlans: requiredPlans,
		allowedPlans:  allowedPlans,
	}

	rs.ValidRules, rs.ValidationInfo = rs.processAndValidate(rulesConfig)
	return rs, err
}

func NewRulesServiceFromSlice(rules []string, allowedPlans sets.Set[string], requiredPlans sets.Set[string]) (*RulesService, error) {

	rulesConfig := &RulesConfig{
		Rules: rules,
	}

	rs := &RulesService{
		parser: &SimpleParser{},
	}

	rs.requiredPlans = requiredPlans
	rs.allowedPlans = allowedPlans
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

	ok, planErrors := validRuleset.checkPlans(rs.allowedPlans, rs.requiredPlans)
	if !ok {
		validationErrors.PlanErrors = append(validationErrors.PlanErrors, planErrors...)
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
	// TODO validate defensively ProvisioningAttributes passed here
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
