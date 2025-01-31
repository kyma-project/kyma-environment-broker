package grammar

import "testing"

import "github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"

func TestParser(t *testing.T) {

    rules.ParserTest(t, &GrammarParser{})
    
}