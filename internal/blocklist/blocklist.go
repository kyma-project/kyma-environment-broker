package blocklist

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// PlanValidator validates plan names. Implemented by broker.AvailablePlansType
// to avoid a circular import.
type PlanValidator interface {
	IsPlanName(name string) bool
}

// Rule holds a parsed blocking rule.
//
// Compact string format: '"message"' or '"message","plan=val1,val2"'
//
// The message is a double-quoted string as the first token. The optional
// second token is a double-quoted "plan=<value>" string where <value> may be
// a comma-separated list of plan names.
//
// The message may contain the {plan} placeholder.
type Rule struct {
	Message string
	Plan    string // empty = match all plans
}

// parseRule parses a compact rule string. Tokens are comma-separated quoted
// strings. The first token is the message; the optional second token is
// "plan=<value>". Any key other than "plan" is an error.
//
//	'"message"'
//	'"message","plan=aws"'
//	'"message","plan=aws,gcp"'
func parseRule(s string) (Rule, error) {
	if strings.TrimSpace(s) == "" {
		return Rule{}, nil // empty string is a no-op, caller must skip
	}
	tokens, err := splitQuotedTokens(s)
	if err != nil {
		return Rule{}, fmt.Errorf("invalid rule %q: %w", s, err)
	}
	if len(tokens) == 0 {
		return Rule{}, fmt.Errorf("empty rule")
	}
	if tokens[0] == "" {
		return Rule{}, fmt.Errorf("empty message in rule %q", s)
	}

	r := Rule{Message: tokens[0]}
	if len(tokens) == 1 {
		return Rule{}, nil // no plan filter — no-op, caller must skip
	}
	for _, tok := range tokens[1:] {
		idx := strings.IndexByte(tok, '=')
		if idx == -1 {
			return Rule{}, fmt.Errorf("invalid key=value token %q in rule %q", tok, s)
		}
		key := strings.TrimSpace(tok[:idx])
		val := strings.TrimSpace(tok[idx+1:])
		if key != "plan" {
			return Rule{}, fmt.Errorf("unknown key %q in rule %q (only \"plan\" is allowed)", key, s)
		}
		if val == "" {
			return Rule{}, fmt.Errorf("empty plan filter in rule %q", s)
		}
		for _, p := range strings.Split(val, ",") {
			if strings.TrimSpace(p) == "" {
				return Rule{}, fmt.Errorf("empty plan segment in rule %q", s)
			}
		}
		r.Plan = val
	}
	return r, nil
}

// splitQuotedTokens splits a string into tokens separated by commas that are
// outside double-quoted strings. Each token has its surrounding quotes stripped.
//
// Example: '"hello","plan=aws,gcp"' → ["hello", "plan=aws,gcp"]
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
			if s == "" {
				return nil, fmt.Errorf("trailing comma in rule")
			}
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
			if r.Message == "" {
				continue // empty string or message-only rule is a no-op
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
	if r.Message == "" {
		*rl = nil
		return nil // empty string or message-only rule is a no-op
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
// It also validates all plan names in rules against the validator, returning an error
// for any unrecognised plan name (e.g. typos like "trail" instead of "trial").
func (b OperationBlocklist) WithPlanValidator(v PlanValidator) (OperationBlocklist, error) {
	b.planValidator = v
	type opRules struct {
		name  string
		rules ruleList
	}
	for _, op := range []opRules{
		{"provision", b.Provision},
		{"update", b.Update},
		{"planUpgrade", b.PlanUpgrade},
		{"deprovision", b.Deprovision},
	} {
		for _, r := range op.rules {
			if r.Plan == "" {
				continue
			}
			for _, p := range strings.Split(r.Plan, ",") {
				p = strings.TrimSpace(p)
				if !v.IsPlanName(p) {
					return OperationBlocklist{}, fmt.Errorf("unknown plan name %q in %s rule", p, op.name)
				}
			}
		}
	}
	return b, nil
}

// ReadFromFile loads an OperationBlocklist from a YAML file.
// The file contains the blocklist fields directly (no outer key):
//
//	provision:
//	  - '"message","plan=trial"'
//
// Unknown top-level keys are rejected to catch typos (e.g. "planUpgarde").
func ReadFromFile(path string) (OperationBlocklist, error) {
	f, err := os.Open(path)
	if err != nil {
		return OperationBlocklist{}, fmt.Errorf("while reading operation blocklist: %w", err)
	}
	defer func() { _ = f.Close() }()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var bl OperationBlocklist
	if err := dec.Decode(&bl); err != nil {
		if errors.Is(err, io.EOF) {
			return OperationBlocklist{}, nil
		}
		return OperationBlocklist{}, fmt.Errorf("while reading operation blocklist: %w", err)
	}
	return bl, nil
}

// CheckProvision returns a non-nil error when a provision rule matches planName.
func (b *OperationBlocklist) CheckProvision(planName string) error {
	return checkRules(b.Provision, b.planValidator, planName)
}

// CheckUpdate returns a non-nil error when an update rule matches planName.
func (b *OperationBlocklist) CheckUpdate(planName string) error {
	return checkRules(b.Update, b.planValidator, planName)
}

// CheckPlanUpgrade returns a non-nil error when a planUpgrade rule matches planName.
func (b *OperationBlocklist) CheckPlanUpgrade(planName string) error {
	return checkRules(b.PlanUpgrade, b.planValidator, planName)
}

// CheckDeprovision returns a non-nil error when a deprovision rule matches planName.
func (b *OperationBlocklist) CheckDeprovision(planName string) error {
	return checkRules(b.Deprovision, b.planValidator, planName)
}

// checkRules iterates rules and returns an error for the first matching one.
func checkRules(rules []Rule, pv PlanValidator, planName string) error {
	for _, r := range rules {
		if matchesRule(r, pv, planName) {
			return fmt.Errorf("%s", formatMessage(r.Message, planName))
		}
	}
	return nil
}

// matchesRule returns true when the rule's plan filter (if any) matches planName.
func matchesRule(r Rule, pv PlanValidator, planName string) bool {
	if r.Plan == "" {
		return true
	}
	return matchesPlan(pv, r.Plan, planName)
}

// matchesPlan checks whether rulePlan (comma-separated list) contains operationPlan.
// When a PlanValidator is set, only recognised plan names can match.
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

// formatMessage replaces {plan} placeholder with the actual plan name.
func formatMessage(msg, planName string) string {
	return strings.ReplaceAll(msg, "{plan}", planName)
}
