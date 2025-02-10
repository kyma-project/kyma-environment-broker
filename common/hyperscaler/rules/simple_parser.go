package rules

import (
	"fmt"
	"strings"
)

type SimpleParser struct{
    
}

func (g* SimpleParser) Parse(ruleEntry string) (*Rule, error) {
    outputRule := &Rule{}

    ruleEntry = strings.ReplaceAll(ruleEntry, " ", "")
    ruleEntry = strings.ReplaceAll(ruleEntry, "\t", "")
    ruleEntry = strings.ReplaceAll(ruleEntry, "\n", "")
    ruleEntry = strings.ReplaceAll(ruleEntry, "\r", "")
    ruleEntry = strings.ReplaceAll(ruleEntry, "\f", "")

    outputInputPart := strings.Split(ruleEntry, "->")

    if len(outputInputPart) > 2 { 
        return nil, fmt.Errorf("rule has more than one arrows")
    }

    inputPart := outputInputPart[0]

    planAndInputAttr := strings.Split(inputPart, "(")

    if len(planAndInputAttr) > 2 {
        return nil, fmt.Errorf("rule has more than one '('")
    }

    forValidationOnly := strings.Split(inputPart, ")")

    if len(forValidationOnly) > 2 {
        return nil, fmt.Errorf("rule has more than one ')'")
    }

    if strings.Contains(inputPart, "(") && !strings.Contains(inputPart, ")") {
        return nil, fmt.Errorf("rule has unclosed parantheses")
    }

    if !strings.Contains(inputPart, "(") && strings.Contains(inputPart, ")") {
        return nil, fmt.Errorf("rule has unclosed parantheses")
    }

    _, err := outputRule.SetPlan(planAndInputAttr[0])
    if err != nil {
        return nil, err
    }

    if len(planAndInputAttr) > 1 {
        inputPart := strings.TrimSuffix(planAndInputAttr[1], ")")

        inputAttrs := strings.Split(inputPart, ",")

        for _, inputAttr := range inputAttrs {

            if inputAttr == "" {
                return nil, fmt.Errorf("input attribute is empty")
            }

            attribute := strings.Split(inputAttr, "=")

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
        outputAttrs := strings.Split(outputInputPart[1], ",")
    
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


