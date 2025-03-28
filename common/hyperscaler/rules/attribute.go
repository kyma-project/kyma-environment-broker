package rules

type Attribute struct {
	Name          string
	Description   string
	Setter        func(*Rule, string) (*Rule, error)
	Getter        func(*Rule) string
	input         bool
	output        bool
	modifiedLabel string
	HasValue      bool

	modifiedLabelName string
}

func (a Attribute) HasLiteral(rule *Rule) bool {
	value := a.Getter(rule)
	return value != ""
}
