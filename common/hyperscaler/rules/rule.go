package rules

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Attribute struct {
	Name          string
	Description   string
	setter        func(*Rule, string) (*Rule, error)
	getter        func(*Rule) string
	input         bool
	output        bool
	modifiedLabel string

	modifiedLabelName string
	ApplyLabel        func(r *Rule, labels map[string]string) map[string]string
}

func (a Attribute) String(r *Rule) any {
	val := a.getter(r)
	output := ""
	if val == "true" {
		output += fmt.Sprintf("%s, ", a.Name)
	} else if val != "" && val != "false" {
		output += fmt.Sprintf("%s=%s, ", a.Name, val)
	}

	return output
}

const (
	PR_ATTR_NAME = "PR"
	HR_ATTR_NAME = "HR"
	EU_ATTR_NAME = "EU"
	S_ATTR_NAME  = "S"

	HYPERSCALER_LABEL = "hyperscalerType"
	EUACCESS_LABEL    = "euAccess"
	SHARED_LABEL      = "shared"

	ASTERISK = "*"
)

var InputAttributes = []Attribute{
	{
		Name:        PR_ATTR_NAME,
		Description: "Platform Region",
		setter:      setPlatformRegion,
		getter:      func(r *Rule) string { return r.PlatformRegion },
		input:       true,
		output:      false,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			if labels[HYPERSCALER_LABEL] == "" {
				labels[HYPERSCALER_LABEL] = r.Plan
			} else {
				labels[HYPERSCALER_LABEL] += "_" + r.PlatformRegion
			}
			return labels
		},
	},
	{
		Name:        HR_ATTR_NAME,
		Description: "Hyperscaler Region",
		setter:      setHyperscalerRegion,
		getter:      func(r *Rule) string { return r.HyperscalerRegion },
		input:       true,
		output:      false,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			if labels[HYPERSCALER_LABEL] == "" {
				labels[HYPERSCALER_LABEL] = r.Plan
			} else {
				labels[HYPERSCALER_LABEL] += "_" + r.HyperscalerRegion
			}
			return labels
		},
	},
}

var OutputAttributes = []Attribute{
	{
		Name:        EU_ATTR_NAME,
		Description: "EU Access",
		setter:      setEuAccess,
		getter:      func(r *Rule) string { return strconv.FormatBool(r.EuAccess) },
		input:       false,
		output:      true,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			if r.EuAccess {
				labels[EUACCESS_LABEL] = "true"
			}
			return labels
		},
	},
	{
		Name:        S_ATTR_NAME,
		Description: "Shared",
		setter:      setShared,
		getter:      func(r *Rule) string { return strconv.FormatBool(r.Shared) },
		input:       false,
		output:      true,
		ApplyLabel: func(r *Rule, labels map[string]string) map[string]string {
			if r.Shared {
				labels[SHARED_LABEL] = "true"
			}

			return labels
		},
	},
}

type Labels struct {
	Labels []string
}

type Rule struct {
	Plan                     string
	PlatformRegion           string
	HyperscalerRegion        string
	EuAccess                 bool
	Shared                   bool
	Labels                   map[string]string
	// Attributes               []Attribute
	ContainsInputAttributes  bool
	ContainsOutputAttributes bool
	hyperscalerNameMappingFunction func(string) string
}

var AllAttributes = append(InputAttributes, OutputAttributes...)

func NewRule() *Rule {
	return &Rule{
		hyperscalerNameMappingFunction: getHyperscalerName,
		Labels: 					  make(map[string]string),
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
	// labels := make(map[string]string)

	for _, attr := range AllAttributes {
		if attr.getter(r) != "" {
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
	return r.Plan == attributes.Plan &&
		(r.PlatformRegion == attributes.PlatformRegion || r.PlatformRegion == ASTERISK) &&
		(r.HyperscalerRegion == attributes.HyperscalerRegion || r.HyperscalerRegion == ASTERISK)
}

func (r *Rule) SetAttributeValue(attribute, value string) (*Rule, error) {
	for _, attr := range AllAttributes {
		if attr.Name == attribute {
			return attr.setter(r, value)
		}
	}

	return nil, fmt.Errorf("unknown attribute %s", attribute)
}

func setShared(r *Rule, value string) (*Rule, error) {
	if r.Shared {
		return nil, fmt.Errorf("Shared already set")
	}

	r.ContainsOutputAttributes = true
	r.Shared = true

	return r, nil
}

func setEuAccess(r *Rule, value string) (*Rule, error) {
	if r.EuAccess {
		return nil, fmt.Errorf("EuAccess already set")
	}
	r.ContainsOutputAttributes = true
	r.EuAccess = true

	return r, nil
}

func (r *Rule) SetPlan(value string) (*Rule, error) {
	if value == "" {
		return nil, fmt.Errorf("plan is empty")
	}

	// validate that the plan is supported
	_, ok := broker.PlanIDsMapping[value]
	if !ok {
		return nil, fmt.Errorf("plan %s is not supported", value)
	}

	r.Plan = value
	r.Labels[HYPERSCALER_LABEL] = r.hyperscalerNameMappingFunction(value)

	return r, nil
}

func setPlatformRegion(r *Rule, value string) (*Rule, error) {
	if r.PlatformRegion != "" {
		return nil, fmt.Errorf("PlatformRegion already set")
	} else if value == "" {
		return nil, fmt.Errorf("PlatformRegion is empty")
	}

	r.ContainsInputAttributes = true
	r.PlatformRegion = value

	return r, nil
}

func setHyperscalerRegion(r *Rule, value string) (*Rule, error) {
	if r.HyperscalerRegion != "" {
		return nil, fmt.Errorf("HyperscalerRegion already set")
	} else if value == "" {
		return nil, fmt.Errorf("HyperscalerRegion is empty")
	}

	r.ContainsInputAttributes = true
	r.HyperscalerRegion = value

	return r, nil
}

func (r *Rule) NumberOfInputAttributes() int {
	count := 0

	for _, attr := range AllAttributes {
		if attr.input {
			count++
		}
	}

	if r.PlatformRegion != "" {
		count++
	}

	if r.HyperscalerRegion != "" {
		count++
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

	labelsStr = labelsStr[:len(labelsStr)-2]

	return fmt.Sprintf("%-50s %-50s", ruleStr, labelsStr)
}

func (r *Rule) StringNoLabels() string {
	ruleStr := fmt.Sprintf("%s", r.Plan)

	if r.ContainsInputAttributes {

		ruleStr += fmt.Sprintf("(")

		for _, attr := range InputAttributes {
			attrStr := attr.String(r)
			ruleStr += fmt.Sprintf("%s", attrStr)
		}

		ruleStr = ruleStr[:len(ruleStr)-2]
		ruleStr += fmt.Sprintf(")")
	}

	if r.ContainsOutputAttributes {
		ruleStr += fmt.Sprintf("-> ")

		for _, attr := range OutputAttributes {
			attrStr := attr.String(r)
			ruleStr += fmt.Sprintf("%s", attrStr)
		}

		ruleStr = ruleStr[:len(ruleStr)-2]
	}

	return ruleStr
}

func (r *Rule) IsResolved() bool {
	return r.Plan != "" && r.PlatformRegion != ASTERISK && r.HyperscalerRegion != ASTERISK
}

func (r *Rule) Combine(rule Rule) *Rule {
	newRule := NewRule()
	newRule.SetPlan(r.Plan)

	if r.PlatformRegion != "" && r.PlatformRegion != ASTERISK {
		newRule.PlatformRegion = r.PlatformRegion
		newRule.ContainsInputAttributes = true
	} else {
		newRule.PlatformRegion = rule.PlatformRegion
		newRule.ContainsInputAttributes = true
	}

	if r.HyperscalerRegion != "" && r.HyperscalerRegion != ASTERISK {
		newRule.HyperscalerRegion = r.HyperscalerRegion
		newRule.ContainsInputAttributes = true
	} else {
		newRule.HyperscalerRegion = rule.HyperscalerRegion
		newRule.ContainsInputAttributes = true
	}

	newRule.hyperscalerNameMappingFunction = r.hyperscalerNameMappingFunction

	return newRule
}
