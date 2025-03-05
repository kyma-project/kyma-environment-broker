package rules

import (
	"fmt"
	"sort"
)

type Labels struct {
	Labels []string
}

type Rule struct {
	Plan                           string
	PlatformRegion                 string
	HyperscalerRegion              string
	EuAccess                       bool
	Shared                         bool
	Labels                         map[string]string
	ContainsInputAttributes        bool
	ContainsOutputAttributes       bool
	hyperscalerNameMappingFunction func(string) string
}

func NewRule() *Rule {
	return &Rule{
		hyperscalerNameMappingFunction: getHyperscalerName,
		Labels:                         make(map[string]string),
	}
}

type MatchableAttributes struct {
	Plan              string `json:"plan"`
	PlatformRegion    string `json:"platformRegion"`
	HyperscalerRegion string `json:"hyperscalerRegion"`
}

func (r *Rule) CalculateLabels() map[string]string {
	return r.CalculateLabelsWith(getHyperscalerName(r.Plan))
}

func (r *Rule) CalculateLabelsWith(hyperscalerName string) map[string]string {
	for _, attr := range AllAttributes {
		if attr.Getter(r) != "" {
			r.Labels = attr.ApplyLabel(r, r.Labels)
		}
	}

	return r.Labels
}

func getHyperscalerName(plan string) (result string) {
	if plan == "aws" || plan == "gcp" || plan == "azure" || plan == "azure_lite" {
		return plan
	} else if plan == "trial" {
		return "aws"
	} else if plan == "free" {
		return "aws/azure"
	} else if plan == "sap-converged-cloud" {
		return "openstack"
	} else if plan == "preview" {
		return "aws"
	} else {
		return ""
	}
}

func (r *Rule) Matched(attributes *MatchableAttributes) bool {

	if r.Plan != attributes.Plan {
		return false
	}

	matched := true
	for _, attr := range InputAttributes {
		value := attr.Getter(r)
		matchableValue := attr.MatchableGetter(attributes)
		matched = matched && (value == matchableValue || value == ASTERISK || value == "")
	}

	return matched
}

func (r *Rule) SetAttributeValue(attribute, value string) (*Rule, error) {
	for _, attr := range AllAttributes {
		if attr.Name == attribute {
			return attr.Setter(r, value)
		}
	}

	return nil, fmt.Errorf("unknown attribute %s", attribute)
}

func (r *Rule) NumberOfInputAttributes() int {
	count := 0

	for _, attr := range InputAttributes {
		value := attr.Getter(r)
		if value != "" {
			count++
		}
	}

	return count
}

func (r *Rule) String() string {
	ruleStr := r.StringNoLabels()

	labels := r.CalculateLabels()
	labelsStr := "# "
	labelsToSort := make([]string, 0, len(labels))
	for key, value := range labels {
		labelsToSort = append(labelsToSort, fmt.Sprintf("%s: %s", key, string(value)))
	}

	sort.Strings(labelsToSort)

	for _, key := range labelsToSort {
		labelsStr += fmt.Sprintf("%s, ", key)
	}

	// remove the last ", "
	labelsStr = labelsStr[:len(labelsStr)-2]

	return fmt.Sprintf("%-50s %-50s", ruleStr, labelsStr)
}

func (r *Rule) StringNoLabels() string {
	ruleStr := fmt.Sprintf("%s", r.Plan)

	if r.ContainsInputAttributes {

		ruleStr += fmt.Sprintf("(")
		ruleStr = r.append(ruleStr, InputAttributes)
		ruleStr += fmt.Sprintf(")")
	}

	if r.ContainsOutputAttributes {
		ruleStr += fmt.Sprintf("-> ")
		ruleStr = r.append(ruleStr, OutputAttributes)
	}

	return ruleStr
}

func (r* Rule) append(ruleStr string, attributes []Attribute) string {
	
	for _, attr := range attributes {
		attrStr := attr.String(r)
		ruleStr += fmt.Sprintf("%s", attrStr)
	}

	// remove the last ", "
	ruleStr = ruleStr[:len(ruleStr)-2]

	return ruleStr
}

func (r *Rule) IsResolved() bool {
	resolved := true

	for _, attr := range InputAttributes {
		value := attr.Getter(r)
		resolved = resolved && value != ASTERISK
	}

	return resolved
}

func (r *Rule) Combine(rule Rule) *Rule {
	newRule := NewRule()
	newRule.SetPlanNoValidation(r.Plan)

	for _, attr := range InputAttributes {
		value := attr.Getter(r)
		if value != "" && value != ASTERISK {
			attr.Setter(newRule, value)
			newRule.ContainsInputAttributes = true
		} else {
			valueR := attr.Getter(&rule)
			attr.Setter(newRule, valueR)
			newRule.ContainsInputAttributes = true
		}
	}

	newRule.hyperscalerNameMappingFunction = r.hyperscalerNameMappingFunction

	return newRule
}

func (r *Rule) SignatureWithValues() string {
	key := r.Plan

	for _, attr := range InputAttributes {
		key += attr.Name + SIGNATURE_ATTR_SEPARATOR
		checkValue := attr.Getter(r)
		key += getAttrValueSymbol(checkValue, ASTERISK, checkValue)
	}

	return key
}

func (r *Rule) MirroredSignature() string {
	return r.SignaturePreKeys(ATTRIBUTE_WITH_VALUE, ASTERISK)
}

func (r *Rule) SignaturePreKeys(positiveKey, mirroredKey string) string {
	signatureKey := r.Plan

	for _, attr := range InputAttributes {
		signatureKey += attr.Name + SIGNATURE_ATTR_SEPARATOR
		checkValue := attr.Getter(r)
		signatureKey += getAttrValueSymbol(checkValue, positiveKey, mirroredKey)
	}

	return signatureKey
}

func getAttrValueSymbol(checkedValue, returnedValueTrue, returnedValueFalse string) string {
	if checkedValue == "" || checkedValue == ASTERISK {
		return returnedValueTrue
	} else {
		return returnedValueFalse
	}

}
