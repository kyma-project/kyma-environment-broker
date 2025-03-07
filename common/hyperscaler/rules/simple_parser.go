package rules

import (
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

const (
	COMMA   = ","
	ARROW   = "->"
	L_PAREN = "("
	R_PAREN = ")"
	EQUAL   = "="
)

type SimpleParser struct {
	enabledPlans *broker.EnablePlans
}

func (g *SimpleParser) Parse(ruleEntry string) (*Rule, error) {
	outputRule := NewRule()

	ruleEntry = RemoveWhitespaces(ruleEntry)

	outputInputPart := strings.Split(ruleEntry, ARROW)

	if len(outputInputPart) > 2 {
		return nil, fmt.Errorf("rule has more than one arrows")
	}

	inputPart := outputInputPart[0]

	planAndInputAttr := strings.Split(inputPart, L_PAREN)

	if len(planAndInputAttr) > 2 {
		return nil, fmt.Errorf("rule has more than one " + L_PAREN)
	}

	forValidationOnly := strings.Split(inputPart, R_PAREN)

	if len(forValidationOnly) > 2 {
		return nil, fmt.Errorf("rule has more than one " + R_PAREN)
	}

	if strings.Contains(inputPart, L_PAREN) && !strings.Contains(inputPart, R_PAREN) {
		return nil, fmt.Errorf("rule has unclosed parentheses")
	}

	if !strings.Contains(inputPart, L_PAREN) && strings.Contains(inputPart, R_PAREN) {
		return nil, fmt.Errorf("rule has unclosed parentheses")
	}

	_, err := outputRule.SetPlan(planAndInputAttr[0], g.enabledPlans)
	if err != nil {
		return nil, err
	}

	if len(planAndInputAttr) > 1 {
		inputPart := strings.TrimSuffix(planAndInputAttr[1], R_PAREN)

		inputAttrs := strings.Split(inputPart, COMMA)

		for _, inputAttr := range inputAttrs {

			if inputAttr == "" {
				return nil, fmt.Errorf("input attribute is empty")
			}

			attribute := strings.Split(inputAttr, EQUAL)

			if len(attribute) != 2 {
				return nil, fmt.Errorf("input attribute has no value")
			}

			_, err := outputRule.SetAttributeValue(attribute[0], attribute[1])
			if err != nil {
				return nil, err
			}
		}
	}

	if len(outputInputPart) > 1 {
		outputAttrs := strings.Split(outputInputPart[1], COMMA)

		for _, outputAttr := range outputAttrs {
			if outputAttr == "" {
				return nil, fmt.Errorf("output attribute is empty")
			}

			_, err := outputRule.SetAttributeValue(outputAttr, "true")
			if err != nil {
				return nil, err
			}
		}
	}

	return outputRule, nil
}
