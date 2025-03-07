package rules

import "fmt"

type Attribute struct {
	Name            string
	Description     string
	Setter          func(*Rule, string) (*Rule, error)
	Getter          func(*Rule) string
	MatchableGetter func(*ProvisioningAttributes) string
	input           bool
	output          bool
	modifiedLabel   string
	HasValue        bool

	modifiedLabelName string
	ApplyLabel        func(r *Rule, labels map[string]string) map[string]string
}

func (a Attribute) HasLiteral(rule *Rule) bool {
	value := a.Getter(rule)
	return value != "*" && value != ""
}

func (a Attribute) String(r *Rule) any {
	val := a.Getter(r)
	output := ""
	if val == "true" {
		output += fmt.Sprintf("%s, ", a.Name)
	} else if val != "" && val != "false" {
		output += fmt.Sprintf("%s=%s, ", a.Name, val)
	}

	return output
}
