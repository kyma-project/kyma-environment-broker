package rules

import (
	"fmt"
)

type Rule struct {
	Plan                     string
	PlatformRegion           string
	PlatformRegionSuffix     bool
	HyperscalerRegionSuffix  bool
	HyperscalerRegion        string
	EuAccess                 bool
	Shared                   bool
	ContainsInputAttributes  bool
	ContainsOutputAttributes bool
}

func NewRule() *Rule {
	return &Rule{}
}

type ProvisioningAttributes struct {
	Plan              string `json:"plan"`
	PlatformRegion    string `json:"platformRegion"`
	HyperscalerRegion string `json:"hyperscalerRegion"`
	Hyperscaler       string `json:"hyperscaler"`
}

func (r *Rule) SetAttributeValue(attribute, value string, attributes []Attribute) error {
	for _, attr := range attributes {
		if attr.Name == attribute {
			return attr.Setter(r, value)
		}
	}

	return fmt.Errorf("unknown attribute %s", attribute)
}
