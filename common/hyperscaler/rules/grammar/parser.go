package grammar

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	parser "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/grammar/antlr"
)

type GrammarParser struct{
    
}

func (g* GrammarParser) Parse(ruleEntry string) *rules.Rule {
		// Setup the input
		is := antlr.NewInputStream(ruleEntry)

		// Create the Lexer
		lexer := parser.NewRuleLexer(is)
		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

		// Create the Parser
		p := parser.NewRuleParserParser(stream)

		// Finally parse the expression
        listener := &RuleListener{processed: &rules.Rule{}}
		antlr.ParseTreeWalkerDefault.Walk(listener, p.RuleEntry())
        return listener.processed
}



