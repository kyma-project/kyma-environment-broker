package rules

import (
	"fmt"
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

func (r *Rule) SetAttributeValue(attribute, value string, attributes []Attribute) (*Rule, error) {
	for _, attr := range attributes {
		if attr.Name == attribute {
			return attr.Setter(r, value)
		}
	}

	return nil, fmt.Errorf("unknown attribute %s", attribute)
}
