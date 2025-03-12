package rules

import (
	"fmt"
	"log"
	"sort"
)

type Labels struct {
	Labels []string
}

type Rule struct {
	Plan                           string
	PlatformRegion                 string
	PlatformRegionSuffix           bool
	HyperscalerRegionSuffix        bool
	HyperscalerRegion              string
	EuAccess                       bool
	Shared                         bool
	ContainsInputAttributes        bool
	ContainsOutputAttributes       bool
	hyperscalerNameMappingFunction func(string) string
}

func NewRule() *Rule {
	return &Rule{
		hyperscalerNameMappingFunction: getHyperscalerName,
	}
}

type ProvisioningAttributes struct {
	Plan              string `json:"plan"`
	PlatformRegion    string `json:"platformRegion"`
	HyperscalerRegion string `json:"hyperscalerRegion"`
	Hyperscaler       string `json:"hyperscaler"`
}

/*
LabelsWithCalculatedHyperscaler calulactes the labels for the rule instead of using ProvisioningAttributes field.
In KEB CalculateLabels must be used
*/
func (r *Rule) LabelsWithCalculatedHyperscaler(provisioningAttributes *ProvisioningAttributes) map[string]string {
	return r.calculateLabels(getHyperscalerName(r.Plan), provisioningAttributes)
}

func (r *Rule) Labels(provisioningAttributes *ProvisioningAttributes) map[string]string {
	return r.calculateLabels(provisioningAttributes.Hyperscaler, provisioningAttributes)
}

func (r *Rule) calculateLabels(hyperscalerName string, provisioningAttributes *ProvisioningAttributes) map[string]string {
	labels := map[string]string{
		HYPERSCALER_LABEL: hyperscalerName,
	}

	for _, attr := range OutputAttributes {
		if attr.Getter(r) != "" {
			labels = attr.ApplyLabel(r, provisioningAttributes, labels)
		}
	}

	return labels
}

func (r *Rule) LabelsNew(provisioningAttributes *ProvisioningAttributes) map[string]string {
	labels := map[string]string{
		HYPERSCALER_LABEL: r.hyperscalerNameMappingFunction(r.Plan),
	}

	if r.EuAccess {
		labels[EUACCESS_LABEL] = "true"
	}

	if r.Shared {
		labels[SHARED_LABEL] = "true"
	}

	if r.PlatformRegionSuffix {
		labels[HYPERSCALER_LABEL] += "_" + provisioningAttributes.PlatformRegion
	}

	if r.HyperscalerRegionSuffix {
		labels[HYPERSCALER_LABEL] += "_" + provisioningAttributes.HyperscalerRegion
	}

	return labels
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

func (r *Rule) Matched(attributes *ProvisioningAttributes) bool {

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

func (r *Rule) SetAttributeValue(attribute, value string, attributes []Attribute) (*Rule, error) {
	for _, attr := range attributes {
		if attr.Name == attribute {
			return attr.Setter(r, value)
		}
	}

	return nil, fmt.Errorf("unknown attribute %s", attribute)
}

func (r *Rule) NumberOfNonEmptyInputAttributes() int {
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

	labels := r.LabelsWithCalculatedHyperscaler(&ProvisioningAttributes{})
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

func (r *Rule) append(ruleStr string, attributes []Attribute) string {

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
	_, err := newRule.SetPlanNoValidation(r.Plan)
	if err != nil {
		log.Panicf("unexpected error while setting a plan : %v", err)
	}

	for _, attr := range InputAttributes {
		value := attr.Getter(r)
		if value != "" && value != ASTERISK {
			_, err = attr.Setter(newRule, value)
			if err != nil {
				log.Panicf("unexpected error while setting a plan : %v", err)
			}
			newRule.ContainsInputAttributes = true
		} else {
			valueR := attr.Getter(&rule)
			_, err := attr.Setter(newRule, valueR)
			if err != nil {
				log.Panicf("unexpected error while setting a plan : %v", err)
			}
			newRule.ContainsInputAttributes = true
		}
	}

	newRule.hyperscalerNameMappingFunction = r.hyperscalerNameMappingFunction

	return newRule
}

func (r *Rule) SignatureWithValues() string {
	signature := r.Plan

	for _, attr := range InputAttributes {
		signature += attr.Name + SIGNATURE_ATTR_SEPARATOR
		checkValue := attr.Getter(r)
		signature += getAttrValueSymbol(checkValue, ASTERISK, checkValue)
	}

	return signature
}

func (r *Rule) MirroredSignature() string {
	return r.SignatureWithSymbols(ATTRIBUTE_WITH_VALUE, ASTERISK)
}

func (r *Rule) SignatureWithSymbols(positiveKey, mirroredKey string) string {
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
