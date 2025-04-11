package rules

import (
	"fmt"
	"strings"
)

const (
	Comma  = ","
	Arrow  = "->"
	LParen = "("
	RParen = ")"
	Equal  = "="
)

type SimpleParser struct {
}

func (g *SimpleParser) Parse(ruleEntry string) (*Rule, error) {
	outputRule := NewRule()

	ruleEntry = RemoveWhitespaces(ruleEntry)

	ruleParts := strings.Split(ruleEntry, Arrow)

	if len(ruleParts) > 2 {
		return nil, fmt.Errorf("rule has more than one arrow")
	}

	leftPart := ruleParts[0]

	planAndInputAttr := strings.Split(leftPart, LParen)

	if len(planAndInputAttr) > 2 {
		return nil, fmt.Errorf("rule has more than one " + LParen)
	}

	forValidationOnly := strings.Split(leftPart, RParen)

	if len(forValidationOnly) > 2 {
		return nil, fmt.Errorf("rule has more than one " + RParen)
	}

	if strings.Contains(leftPart, LParen) && !strings.Contains(leftPart, RParen) {
		return nil, fmt.Errorf("rule has not balanced parentheses")
	}

	if !strings.Contains(leftPart, LParen) && strings.Contains(leftPart, RParen) {
		return nil, fmt.Errorf("rule has not balanced parentheses")
	}

	_, err := outputRule.SetPlan(planAndInputAttr[0])
	if err != nil {
		return nil, err
	}

	if len(planAndInputAttr) > 1 {

		inputPart := strings.TrimSuffix(planAndInputAttr[1], RParen)

		inputAttrs := strings.Split(inputPart, Comma)

		for _, inputAttr := range inputAttrs {

			if inputAttr == "" {
				return nil, fmt.Errorf("input attribute is empty")
			}

			attribute := strings.Split(inputAttr, Equal)

			if len(attribute) != 2 {
				return nil, fmt.Errorf("input attribute has no value")
			}

			err = outputRule.SetAttributeValue(attribute[0], attribute[1], InputAttributes)

			if err != nil {
				return nil, err
			}
		}
	}

	if len(ruleParts) > 1 {
		rightPart := ruleParts[1]
		outputAttrs := strings.Split(rightPart, Comma)

		for _, outputAttr := range outputAttrs {
			if outputAttr == "" {
				return nil, fmt.Errorf("output attribute is empty")
			}

			err = outputRule.SetAttributeValue(outputAttr, "true", OutputAttributes)
			if err != nil {
				return nil, err
			}
		}
	}

	return outputRule, nil
}
