package rules

import "fmt"

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

func (r *ValidRule) toResult(provisioningAttributes *ProvisioningAttributes) Result {
	hyperscalerType := provisioningAttributes.Hyperscaler
	if r.PlatformRegionSuffix {
		hyperscalerType += "_" + provisioningAttributes.PlatformRegion
	}
	if r.HyperscalerRegionSuffix {
		hyperscalerType += "_" + provisioningAttributes.HyperscalerRegion
	}

	return Result{
		HyperscalerType: hyperscalerType,
		EUAccess:        r.EuAccess,
		Shared:          r.Shared,
		RawData:         r.RawData,
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
			//TODO consider referring to rule number (line number), referring to both duplicated rules in their raw/unprocessed form
			duplicateErrors = append(duplicateErrors, fmt.Errorf("rule is not unique: %s", rule.keyString()))
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

	//TODO extract preparation
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
					ambiguityErrors = append(ambiguityErrors, fmt.Errorf("rules %s and %s are ambiguous: missing %s", prRule.keyString(), hrRule.keyString(), unionRule.keyString()))
				}
			}
		}
	}

	return len(ambiguityErrors) == 0, ambiguityErrors
}
