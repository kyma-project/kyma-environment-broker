package rules

import (
	"strings"
)

type SimpleParser struct{
    
}

func (g* SimpleParser) Parse(ruleEntry string) *Rule {
    outputRule := &Rule{}

    outputInputPart := strings.Split(ruleEntry, "->")

    inputPart := outputInputPart[0]

    planAndInputAttr := strings.Split(inputPart, "(")

    outputRule.Plan = planAndInputAttr[0]

    if len(planAndInputAttr) > 1 {
        inputPart := strings.TrimSuffix(planAndInputAttr[1], ")")

        inputAttrs := strings.Split(inputPart, ",")

        for _, inputAttr := range inputAttrs {
            if strings.Contains(inputAttr, "PR") {
                outputRule.PlatformRegion = strings.Split(inputAttr, "=")[1]
            }

            if strings.Contains(inputAttr, "HR") {
                outputRule.HyperscalerRegion = strings.Split(inputAttr, "=")[1]
            }
        }

    }

    if len(outputInputPart) > 1 {
        outputAttrs := strings.Split(outputInputPart[1], ",")
    
        for _, outputAttr := range outputAttrs {
            if outputAttr == "S" {
                outputRule.Shared = true
            } else if outputAttr == "EU" {
                outputRule.EuAccess = true
            }
        }
    }

    return outputRule
}
