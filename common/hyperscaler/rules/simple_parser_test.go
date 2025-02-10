package rules

import "testing"

func TestParser(t *testing.T) {
    ParserHappyPathTest(t, &SimpleParser{})
    ParserValidationTest(t, &SimpleParser{})
}