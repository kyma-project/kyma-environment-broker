package rules

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
)

type PatternAttribute struct {
	matchAny bool
	literal  string
}

type RawData struct {
	Rule   string
	RuleNo int
}
type ValidRule struct {
	Plan                    PatternAttribute
	PlatformRegion          PatternAttribute
	HyperscalerRegion       PatternAttribute
	Shared                  bool
	EuAccess                bool
	PlatformRegionSuffix    bool
	HyperscalerRegionSuffix bool
	MatchAnyCount           int
	RawData                 RawData
}

type ValidationErrors struct {
	ParsingErrors   []error
	DuplicateErrors []error
	AmbiguityErrors []error
	PlanErrors      []error
}

func (ve *ValidationErrors) All() []error {
	return append(append(append(ve.ParsingErrors, ve.DuplicateErrors...), ve.AmbiguityErrors...), ve.PlanErrors...)
}

func (vr *ValidRule) Rule() string {
	return vr.RawData.Rule
}

func (vr *ValidRule) NumberedRule() string {
	return vr.RawData.NumberedRule()
}

func (rd *RawData) NumberedRule() string {
	return fmt.Sprintf("%d: %s", rd.RuleNo, rd.Rule)
}

func (pa *PatternAttribute) Match(value string) bool {
	if pa.matchAny {
		return true
	}
	return pa.literal == value
}

func (vr *ValidRule) Match(provisioningAttributes *ProvisioningAttributes) bool {
	if !vr.Plan.Match(provisioningAttributes.Plan) {
		return false
	}

	return vr.matchInputParameters(provisioningAttributes)
}

func (vr *ValidRule) matchInputParameters(provisioningAttributes *ProvisioningAttributes) bool {
	if !vr.PlatformRegion.Match(provisioningAttributes.PlatformRegion) {
		return false
	}

	if !vr.HyperscalerRegion.Match(provisioningAttributes.HyperscalerRegion) {
		return false
	}
	return true
}

func (vr *ValidRule) toResult(provisioningAttributes *ProvisioningAttributes) Result {
	hyperscalerType := provisioningAttributes.Hyperscaler
	if vr.PlatformRegionSuffix {
		hyperscalerType += "_" + provisioningAttributes.PlatformRegion
	}
	if vr.HyperscalerRegionSuffix {
		hyperscalerType += "_" + provisioningAttributes.HyperscalerRegion
	}

	return Result{
		HyperscalerType: hyperscalerType,
		EUAccess:        vr.EuAccess,
		Shared:          vr.Shared,
		RawData:         vr.RawData,
	}
}

type ValidRuleset struct {
	Rules []ValidRule
}

func NewValidRuleset() *ValidRuleset {
	validRules := make([]ValidRule, 0)
	return &ValidRuleset{Rules: validRules}
}

func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		ParsingErrors:   make([]error, 0),
		DuplicateErrors: make([]error, 0),
		AmbiguityErrors: make([]error, 0),
		PlanErrors:      make([]error, 0),
	}
}

func (vr *ValidRule) keyString() string {
	return fmt.Sprintf("%s(PR=%s,HR=%s)", vr.Plan.literal, vr.PlatformRegion.literal, vr.HyperscalerRegion.literal)
}

func (vr *ValidRuleset) checkUniqueness() (bool, []error) {
	uniqueRules := make(map[string]struct{})
	duplicateErrors := make([]error, 0)
	for _, rule := range vr.Rules {
		if _, ok := uniqueRules[rule.keyString()]; ok {
			duplicateErrors = append(duplicateErrors, fmt.Errorf("rule %s is not unique", rule.NumberedRule()))
		} else {
			uniqueRules[rule.keyString()] = struct{}{}
		}
	}

	return len(duplicateErrors) == 0, duplicateErrors
}

// This is 2D solution that does not scale to more than 2 attributes
func (vr *ValidRuleset) checkUnambiguity() (bool, []error) {
	ambiguityErrors := make([]error, 0)

	mostSpecificRules := make(map[string]struct{})
	prSpecified := make([]ValidRule, 0)
	hrSpecified := make([]ValidRule, 0)

	for _, rule := range vr.Rules {
		if rule.MatchAnyCount == 0 {
			mostSpecificRules[rule.keyString()] = struct{}{}
		}
		if rule.MatchAnyCount == 1 {
			if !rule.PlatformRegion.matchAny {
				prSpecified = append(prSpecified, rule)
			} else {
				hrSpecified = append(hrSpecified, rule)
			}
		}
	}

	for _, prRule := range prSpecified {
		for _, hrRule := range hrSpecified {
			if prRule.Plan.literal == hrRule.Plan.literal {
				unionRule := ValidRule{
					Plan:              prRule.Plan,
					PlatformRegion:    prRule.PlatformRegion,
					HyperscalerRegion: hrRule.HyperscalerRegion,
				}
				if _, ok := mostSpecificRules[unionRule.keyString()]; !ok {
					ambiguityErrors = append(ambiguityErrors, fmt.Errorf("rules %s and %s are ambiguous: missing %s", prRule.NumberedRule(), hrRule.NumberedRule(), unionRule.keyString()))
				}
			}
		}
	}

	return len(ambiguityErrors) == 0, ambiguityErrors
}

func (vr *ValidRuleset) checkPlans(allowed sets.Set[string], required sets.Set[string]) (bool, []error) {
	requiredPlans := required.Clone()
	planErrors := make([]error, 0)

	for _, rule := range vr.Rules {
		requiredPlans.Delete(rule.Plan.literal)
		if !allowed.Has(rule.Plan.literal) {
			planErrors = append(planErrors, fmt.Errorf("plan %s is not supported", rule.Plan.literal))
		}
	}
	if requiredPlans.Len() > 0 {
		planErrors = append(planErrors, fmt.Errorf("required plans %v are not covered by rules", requiredPlans))
	}
	return len(planErrors) == 0, planErrors
}

type Result struct {
	HyperscalerType string
	EUAccess        bool
	Shared          bool
	RawData         RawData
}

func (r Result) Hyperscaler() string {
	return r.HyperscalerType
}

func (r Result) IsShared() bool {
	return r.Shared
}

func (r Result) IsEUAccess() bool {
	return r.EUAccess
}

func (r Result) Rule() string {
	return r.RawData.Rule
}

func (r Result) NumberedRule() string {
	return r.RawData.NumberedRule()
}
