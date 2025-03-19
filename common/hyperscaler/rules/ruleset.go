package rules

import "fmt"

type PatternAttribute struct {
	matchAny bool
	literal  string
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
}

type ValidationErrors struct {
	ParsingErrors    []error
	UniquenessErrors []error
	AmbiguityErrors  []error
}

func (pa *PatternAttribute) Match(value string) bool {
	if pa.matchAny {
		return true
	}
	return pa.literal == value
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
		ParsingErrors:    make([]error, 0),
		UniquenessErrors: make([]error, 0),
		AmbiguityErrors:  make([]error, 0),
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

func (vr *ValidRuleset) checkUnambiguity() (bool, []error) {
	ambiguityErrors := make([]error, 0)

	return len(ambiguityErrors) == 0, ambiguityErrors
}
