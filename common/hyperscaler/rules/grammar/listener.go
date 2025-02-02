package grammar

import (
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	parser "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/grammar/antlr"
)

type RuleListener struct {
    *parser.BaseRuleParserListener

    processed *rules.Rule
}

func (r RuleListener) EnterEntry(c *parser.EntryContext) {
	if c.PLAN() != nil {
		r.processed.Plan = c.PLAN().GetText()
	}
}

func (r *RuleListener) EnterPrVal(c *parser.PrValContext) {
	if c.Val() != nil {
		r.processed.PlatformRegion = c.Val().GetText()
	}
}

func (r *RuleListener) EnterHrVal(c *parser.HrValContext) {
	if c.Val() != nil {
		r.processed.HyperscalerRegion = c.Val().GetText()
	}
}

func (s *RuleListener) EnterS(c *parser.SContext) {
	if c.S() != nil {
		s.processed.Shared = true
	}
}

func (s *RuleListener) EnterEu(c *parser.EuContext) {
	if c.EU() != nil {
		s.processed.EuAccess = true
	}
}
