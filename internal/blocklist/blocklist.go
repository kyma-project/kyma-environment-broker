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

// OperationContext carries the attributes of an incoming operation used when
// matching blocking rules. Add fields here to extend filtering capabilities
// (e.g. SubAccountID for SA!= support in the future).
type OperationContext struct {
	PlanName        string
	GlobalAccountID string
}

// Rule holds a parsed blocking rule.
//
// Compact string format: '"message"' or '"message","plan=val1,val2"'
// or '"message","plan=val1,val2","GA!=<id1>,<id2>"'
//
// The message is a double-quoted string as the first token. The optional
// tokens are double-quoted key=value or key!=value strings.
//
// The message may contain the {plan} placeholder.
type Rule struct {
	Message               string
	Plan                  string   // comma-separated list; empty = match all plans
	IncludeGlobalAccounts []string // GA= values; nil = no filter
	ExcludeGlobalAccounts []string // GA!= values; nil = no exclusion
}

// parseRule parses a compact rule string. Tokens are comma-separated quoted
// strings. The first token is the message; optional further tokens are filters.
//
//	'"message"'
//	'"message","plan=aws"'
//	'"message","plan=aws,gcp"'
//	'"message","plan=trial","GA!=12345"'
//	'"message","plan=trial","GA=12345"'
//
// Supported filter tokens:
//   - plan=<name1>,<name2>  — match specific plans (comma-separated)
//   - GA=<globalAccountID>  — match only the specified GlobalAccount
//   - GA!=<globalAccountID> — exclude a GlobalAccount from being blocked
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
		return Rule{}, nil // no filters — no-op, caller must skip
	}
	for _, tok := range tokens[1:] {
		// Check for negation operator (!=) before equality (=) to avoid
		// misparse: "GA!=X" with IndexByte('=') would yield key="GA!".
		if bangIdx := strings.Index(tok, "!="); bangIdx != -1 {
			key := strings.TrimSpace(tok[:bangIdx])
			val := strings.TrimSpace(tok[bangIdx+2:])
			switch key {
			case "GA":
				parts, err := parseGAList(val, s)
				if err != nil {
					return Rule{}, err
				}
				r.ExcludeGlobalAccounts = parts
			default:
				return Rule{}, fmt.Errorf("unknown negation key %q in rule %q", key, s)
			}
			continue
		}
		idx := strings.IndexByte(tok, '=')
		if idx == -1 {
			return Rule{}, fmt.Errorf("invalid key=value token %q in rule %q", tok, s)
		}
		key := strings.TrimSpace(tok[:idx])
		val := strings.TrimSpace(tok[idx+1:])
		switch key {
		case "plan":
			if val == "" {
				return Rule{}, fmt.Errorf("empty plan filter in rule %q", s)
			}
			for _, p := range strings.Split(val, ",") {
				if strings.TrimSpace(p) == "" {
					return Rule{}, fmt.Errorf("empty plan segment in rule %q", s)
				}
			}
			r.Plan = val
		case "GA":
			parts, err := parseGAList(val, s)
			if err != nil {
				return Rule{}, err
			}
			r.IncludeGlobalAccounts = parts
		default:
			return Rule{}, fmt.Errorf("unknown key %q in rule %q (allowed: \"plan=\", \"GA=\", \"GA!=\")", key, s)
		}
	}
	if (len(r.IncludeGlobalAccounts) > 0 || len(r.ExcludeGlobalAccounts) > 0) && r.Plan == "" {
		return Rule{}, fmt.Errorf("GA filter requires plan= in rule %q", s)
	}
	return r, nil
}

// parseGAList splits a comma-separated GA value, trims spaces, and validates
// that no segment is empty. Used for both GA= and GA!= filter tokens.
func parseGAList(val, rule string) ([]string, error) {
	if val == "" {
		return nil, fmt.Errorf("empty GA value in rule %q", rule)
	}
	parts := strings.Split(val, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
		if parts[i] == "" {
			return nil, fmt.Errorf("empty GA segment in rule %q", rule)
		}
	}
	return parts, nil
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

// CheckProvision returns a non-nil error when a provision rule matches ctx.
func (b *OperationBlocklist) CheckProvision(ctx OperationContext) error {
	return checkRules(b.Provision, b.planValidator, ctx)
}

// CheckUpdate returns a non-nil error when an update rule matches ctx.
func (b *OperationBlocklist) CheckUpdate(ctx OperationContext) error {
	return checkRules(b.Update, b.planValidator, ctx)
}

// CheckPlanUpgrade returns a non-nil error when a planUpgrade rule matches ctx.
func (b *OperationBlocklist) CheckPlanUpgrade(ctx OperationContext) error {
	return checkRules(b.PlanUpgrade, b.planValidator, ctx)
}

// CheckDeprovision returns a non-nil error when a deprovision rule matches ctx.
func (b *OperationBlocklist) CheckDeprovision(ctx OperationContext) error {
	return checkRules(b.Deprovision, b.planValidator, ctx)
}

// checkRules iterates rules and returns an error for the first matching one.
func checkRules(rules []Rule, pv PlanValidator, ctx OperationContext) error {
	for _, r := range rules {
		if matchesRule(r, pv, ctx) {
			return fmt.Errorf("%s", formatMessage(r.Message, ctx))
		}
	}
	return nil
}

// matchesRule returns true when all of the rule's filters match the context.
// Each guard returns false when its condition excludes this operation from the rule.
func matchesRule(r Rule, pv PlanValidator, ctx OperationContext) bool {
	if r.Plan != "" && !matchesPlan(pv, r.Plan, ctx.PlanName) {
		return false
	}
	if len(r.IncludeGlobalAccounts) > 0 {
		matched := false
		for _, ga := range r.IncludeGlobalAccounts {
			if strings.EqualFold(ga, ctx.GlobalAccountID) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(r.ExcludeGlobalAccounts) > 0 {
		for _, ga := range r.ExcludeGlobalAccounts {
			if strings.EqualFold(ga, ctx.GlobalAccountID) {
				return false
			}
		}
	}
	return true
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

// formatMessage replaces {plan} and {globalAccount} placeholders.
func formatMessage(msg string, ctx OperationContext) string {
	msg = strings.ReplaceAll(msg, "{plan}", ctx.PlanName)
	msg = strings.ReplaceAll(msg, "{globalAccount}", ctx.GlobalAccountID)
	return msg
}
