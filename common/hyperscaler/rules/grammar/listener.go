package grammar

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	parser "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/grammar/antlr"
)

type RuleListener struct {
    *parser.BaseRuleParserListener

    processed *rules.Rule
}

func (r RuleListener) EnterEntry(c *parser.EntryContext) {
	if c.PLAN() != nil {
		_, err := r.processed.SetPlan(c.PLAN().GetText())
		if err != nil {
			reportError(err.Error(), c.BaseParserRuleContext, c.GetParser())	
		}
	}
}

func (r *RuleListener) EnterPrVal(c *parser.PrValContext) {
	if c.Val() != nil {
		_, err := r.processed.SetAttributeValue("PR", c.Val().GetText())
		if err != nil {
			reportError(err.Error(), c.BaseParserRuleContext, c.GetParser())	
		}
	}
}

func (r *RuleListener) EnterHrVal(c *parser.HrValContext) {
	if c.Val() != nil {
		_, err := r.processed.SetAttributeValue("HR", c.Val().GetText())
		if err != nil {
			reportError(err.Error(), c.BaseParserRuleContext, c.GetParser())	
		}
	}
}



func (r *RuleListener) EnterS(c *parser.SContext) {
	if c.S() != nil {
		_, err := r.processed.SetAttributeValue("S", "true")
		if err != nil {
			reportError(err.Error(), c.BaseParserRuleContext, c.GetParser())	
		}
	}
}

func (r *RuleListener) EnterEu(c *parser.EuContext) {
	if c.EU() != nil {
		_, err := r.processed.SetAttributeValue("EU", "true")	
		if err != nil {
			reportError(err.Error(), c.BaseParserRuleContext, c.GetParser())	
		}
	}
}

func reportError(msg string, ruleCtx antlr.BaseParserRuleContext, parser antlr.Parser) {
	excp := antlr.NewBaseRecognitionException(msg, parser, nil, &ruleCtx)
	ruleCtx.SetException(excp)
	parser.GetErrorHandler().ReportError(parser, excp)
	parser.SetError(excp)
}