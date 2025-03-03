package rules

type Parser interface {
	Parse(ruleEntry string) (*Rule, error)
}
