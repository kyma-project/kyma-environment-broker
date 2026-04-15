package blocklist

import (
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

// PlanValidator resolves and validates plan names. Implemented by the broker's
// AvailablePlansType so the blocklist package avoids a circular import.
type PlanValidator interface {
	// IsPlanName returns true when name is a recognised plan name (case-insensitive).
	IsPlanName(name string) bool
}

// Rule holds a parsed blocking rule.
//
// Compact string format: '"message","key=value","key=value",...'
//
// The message is a double-quoted string as the first token. Each subsequent
// comma-separated token is also a double-quoted "key=value" string.
// Within a quoted value, commas are literal (e.g. "GA=id1,id2").
//
// Allowed keys:
//   - plan  — plan name (e.g. "aws"); comma-separated list supported; matched via PlanValidator
//   - GA    — global account ID list (comma-separated; prefix "!" to negate)
//   - SA    — subaccount ID list (comma-separated; prefix "!" to negate)
//   - HR    — hyperscaler region list (comma-separated; prefix "!" to negate)
//     for provision, uses the caller-supplied region; empty region never matches
//   - PR    — platform region (parsed, not yet checked)
//
// The message may contain {plan}, {GA}, {SA}, {HR}, {PR} placeholders.
type Rule struct {
	Message string
	Params  map[string]string
}

// parseRule parses a compact rule string. Tokens are comma-separated quoted
// strings. The first token is the message; subsequent tokens are key=value pairs.
//
//	'"message","key=val","key=v1,v2"'
func parseRule(s string) (Rule, error) {
	tokens, err := splitQuotedTokens(s)
	if err != nil {
		return Rule{}, fmt.Errorf("invalid rule %q: %w", s, err)
	}
	if len(tokens) == 0 {
		return Rule{}, fmt.Errorf("empty rule")
	}

	message := tokens[0]
	params := make(map[string]string)
	for _, tok := range tokens[1:] {
		idx := strings.IndexByte(tok, '=')
		if idx == -1 {
			return Rule{}, fmt.Errorf("invalid key=value token %q in rule %q", tok, s)
		}
		key := strings.TrimSpace(tok[:idx])
		val := strings.TrimSpace(tok[idx+1:])
		params[key] = val
	}

	return Rule{Message: message, Params: params}, nil
}

// splitQuotedTokens splits a string into tokens separated by commas that are
// outside double-quoted strings. Each token has its surrounding quotes stripped.
//
// Example: '"hello","plan=aws","GA=id1,id2"' → ["hello", "plan=aws", "GA=id1,id2"]
func splitQuotedTokens(s string) ([]string, error) {
	var tokens []string
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		s = strings.TrimSpace(s)
		if s == "" {
			break
		}
		if s[0] != '"' {
			return nil, fmt.Errorf("expected '\"' but got %q", string(s[0]))
		}
		// find the closing quote
		end := strings.Index(s[1:], `"`)
		if end == -1 {
			return nil, fmt.Errorf("unterminated quoted token")
		}
		token := s[1 : end+1]
		tokens = append(tokens, token)
		s = strings.TrimSpace(s[end+2:])
		if len(s) > 0 {
			if s[0] != ',' {
				return nil, fmt.Errorf("expected ',' between tokens but got %q", string(s[0]))
			}
			s = strings.TrimSpace(s[1:])
		}
	}
	return tokens, nil
}

// ruleList is a YAML helper that accepts either a single string or a list of strings.
type ruleList []Rule

func (rl *ruleList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var list []string
	if err := unmarshal(&list); err == nil {
		rules := make([]Rule, 0, len(list))
		for _, s := range list {
			r, err := parseRule(s)
			if err != nil {
				return err
			}
			rules = append(rules, r)
		}
		*rl = rules
		return nil
	}

	var single string
	if err := unmarshal(&single); err != nil {
		return fmt.Errorf("blocklist rule must be a string or list of strings: %w", err)
	}
	r, err := parseRule(single)
	if err != nil {
		return err
	}
	*rl = []Rule{r}
	return nil
}

// OperationBlocklist holds per-operation-type blocking rules.
type OperationBlocklist struct {
	Provision   ruleList `yaml:"provision"`
	Update      ruleList `yaml:"update"`
	PlanUpgrade ruleList `yaml:"planUpgrade"`
	Deprovision ruleList `yaml:"deprovision"`

	planValidator PlanValidator
}

// WithPlanValidator returns a copy of the blocklist with the given PlanValidator set.
func (b OperationBlocklist) WithPlanValidator(v PlanValidator) OperationBlocklist {
	b.planValidator = v
	return b
}

// ReadFromFile loads an OperationBlocklist from a YAML file.
// The file contains the blocklist fields directly (no outer key):
//
//	provision:
//	  - '"message","plan=trial"'
func ReadFromFile(path string) (OperationBlocklist, error) {
	var bl OperationBlocklist
	if err := utils.UnmarshalYamlFile(path, &bl); err != nil {
		return OperationBlocklist{}, fmt.Errorf("while reading operation blocklist: %w", err)
	}
	return bl, nil
}

// CheckProvision returns a non-nil error when a provision rule matches planName, globalAccountID, subAccountID, or hyperscalerRegion.
// hyperscalerRegion is the caller-supplied region; an empty string never matches an HR filter.
func (b *OperationBlocklist) CheckProvision(planName, globalAccountID, subAccountID, hyperscalerRegion string) error {
	return checkRules(b.Provision, b.planValidator, planName, globalAccountID, subAccountID, hyperscalerRegion)
}

// CheckUpdate returns a non-nil error when an update rule matches planName, globalAccountID, subAccountID, or hyperscalerRegion.
func (b *OperationBlocklist) CheckUpdate(planName, globalAccountID, subAccountID, hyperscalerRegion string) error {
	return checkRules(b.Update, b.planValidator, planName, globalAccountID, subAccountID, hyperscalerRegion)
}

// CheckPlanUpgrade returns a non-nil error when a planUpgrade rule matches planName, globalAccountID, subAccountID, or hyperscalerRegion.
func (b *OperationBlocklist) CheckPlanUpgrade(planName, globalAccountID, subAccountID, hyperscalerRegion string) error {
	return checkRules(b.PlanUpgrade, b.planValidator, planName, globalAccountID, subAccountID, hyperscalerRegion)
}

// CheckDeprovision returns a non-nil error when a deprovision rule matches planName, globalAccountID, subAccountID, or hyperscalerRegion.
func (b *OperationBlocklist) CheckDeprovision(planName, globalAccountID, subAccountID, hyperscalerRegion string) error {
	return checkRules(b.Deprovision, b.planValidator, planName, globalAccountID, subAccountID, hyperscalerRegion)
}

// checkRules iterates rules and returns an error for the first matching one.
func checkRules(rules []Rule, pv PlanValidator, planName, globalAccountID, subAccountID, hyperscalerRegion string) error {
	for _, r := range rules {
		if matchesRule(r, pv, planName, globalAccountID, subAccountID, hyperscalerRegion) {
			return fmt.Errorf("%s", formatMessage(r.Message, planName, globalAccountID, subAccountID, hyperscalerRegion))
		}
	}
	return nil
}

// matchesRule returns true when all present filter params of the rule match.
func matchesRule(r Rule, pv PlanValidator, planName, globalAccountID, subAccountID, hyperscalerRegion string) bool {
	if plan, ok := r.Params["plan"]; ok {
		if !matchesPlan(pv, plan, planName) {
			return false
		}
	}
	if ga, ok := r.Params["GA"]; ok {
		if !matchesIDList(ga, globalAccountID) {
			return false
		}
	}
	if sa, ok := r.Params["SA"]; ok {
		if !matchesIDList(sa, subAccountID) {
			return false
		}
	}
	if hr, ok := r.Params["HR"]; ok {
		if !matchesIDList(hr, hyperscalerRegion) {
			return false
		}
	}
	if _, ok := r.Params["PR"]; ok {
		if !matchesPR() {
			return false
		}
	}
	return true
}

// matchesPlan checks whether the rule's plan value matches the operation's plan name.
// rulePlan may be a comma-separated list (e.g. "aws,gcp"); the operation matches if
// any entry in the list matches. When a PlanValidator is available it is used to
// validate each entry so that unknown plan names never match.
// Falls back to case-insensitive string comparison when no validator is set.
func matchesPlan(pv PlanValidator, rulePlan, operationPlan string) bool {
	for _, p := range strings.Split(rulePlan, ",") {
		p = strings.TrimSpace(p)
		if pv != nil {
			if pv.IsPlanName(p) && strings.EqualFold(p, operationPlan) {
				return true
			}
		} else {
			if strings.EqualFold(p, operationPlan) {
				return true
			}
		}
	}
	return false
}

// matchesIDList checks whether value satisfies the id list expression.
// If the expression starts with "!" the match is negated (block if NOT in list).
// IDs within the list are comma-separated (e.g. "id1,id2").
func matchesIDList(expr, value string) bool {
	negate := strings.HasPrefix(expr, "!")
	list := expr
	if negate {
		list = expr[1:]
	}

	inList := false
	for _, id := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(id), value) {
			inList = true
			break
		}
	}

	if negate {
		return !inList
	}
	return inList
}

// matchesHR is implemented via matchesIDList — HR rules use the same comma-separated
// list and negation semantics as GA/SA. An empty hyperscalerRegion never matches.

// matchesPR checks the platform region filter. Not yet implemented.
func matchesPR() bool {
	return true
}

// formatMessage replaces {plan}, {GA}, {SA}, {HR}, {PR} placeholders.
func formatMessage(msg, planName, globalAccountID, subAccountID, hyperscalerRegion string) string {
	msg = strings.ReplaceAll(msg, "{plan}", planName)
	msg = strings.ReplaceAll(msg, "{GA}", globalAccountID)
	msg = strings.ReplaceAll(msg, "{SA}", subAccountID)
	msg = strings.ReplaceAll(msg, "{HR}", hyperscalerRegion)
	return msg
}
