package grammar

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	parser "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules/grammar/antlr"
)

type GrammarParser struct{
}

func (g* GrammarParser) Parse(ruleEntry string) (*rules.Rule, error) {
		is := antlr.NewInputStream(ruleEntry)

		erorrsListener := &ErrorListener{}

		lexer := parser.NewRuleLexer(is)
		lexer.RemoveErrorListeners()
		lexer.AddErrorListener(erorrsListener)

		stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

		p := parser.NewRuleParserParser(stream)
	
		p.RemoveErrorListeners()
		p.AddErrorListener(erorrsListener)

        listener := &RuleListener{processed: &rules.Rule{}}
		antlr.ParseTreeWalkerDefault.Walk(listener, p.RuleEntry())

		if len(erorrsListener.Errors) > 0 {
			return nil, erorrsListener.Errors[0]
		}

        return listener.processed, nil
}

type SyntaxError struct {
    line, column int
    msg          string
}

func (c *SyntaxError) Error() string {
	return c.msg
}

type ErrorListener struct {
    *antlr.DefaultErrorListener
    Errors []error
}

func (c *ErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
    c.Errors = append(c.Errors, &SyntaxError{
        line:   line,
        column: column,
        msg:    msg,
    })
}



