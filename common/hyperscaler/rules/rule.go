package rules

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Rule struct {
	Plan              string
	PlatformRegion    string
	HyperscalerRegion string
	EuAccess          bool
	Shared            bool
}

type MatchableAttributes struct {
	Plan              string `json:"plan"`
	PlatformRegion    string `json:"platformRegion"`
	HyperscalerRegion string `json:"hyperscalerRegion"`
}

func (r *Rule) Labels() []string {
	var result []string = make([]string, 0)
	hyperscalerType := "hyperscalerType: " + getHyperscalerName(r.Plan)
	if r.PlatformRegion != "" {
		hyperscalerType += "_" + r.PlatformRegion
	}
	if r.HyperscalerRegion != "" {
		hyperscalerType += "_" + r.HyperscalerRegion
	}
	result = append(result, hyperscalerType)

	if r.EuAccess {
		result = append(result, "euAccess: true")
	}

	if r.Shared {
		result = append(result, "shared: true")
	}
	return result
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
	return r.Plan == attributes.Plan && r.PlatformRegion == attributes.PlatformRegion && r.HyperscalerRegion == attributes.HyperscalerRegion
}

func (r *Rule) SetAttributeValue(attribute, value string) (*Rule, error) {
	switch attribute {
	case "PR":
		if r.PlatformRegion != "" {
			return nil, fmt.Errorf("PlatformRegion already set")
		} else if value == "" {
			return nil, fmt.Errorf("PlatformRegion is empty")
		}

		r.PlatformRegion = value
	case "HR":
		if r.HyperscalerRegion != "" {
			return nil, fmt.Errorf("HyperscalerRegion already set")
		} else if value == "" {
			return nil, fmt.Errorf("HyperscalerRegion is empty")
		}

		r.HyperscalerRegion = value
	case "EU":
		if r.EuAccess {
			return nil, fmt.Errorf("EuAccess already set")
		}
		r.EuAccess = true
	case "S":
		if r.Shared {
			return nil, fmt.Errorf("Shared already set")
		}

		r.Shared = true
	default:
		return nil, fmt.Errorf("unknown attribute %s", attribute)
	}

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
	return r, nil
}

func (r *Rule) NumberOfInputAtributes() int {
	count := 0

	if r.PlatformRegion != "" {
		count++
	}

	if r.HyperscalerRegion != "" {
		count++
	}

	return count
}

func (r *Rule) String() string {
	ruleStr := fmt.Sprintf("%s", r.Plan)

	if r.PlatformRegion != "" || r.HyperscalerRegion != "" {
		ruleStr += fmt.Sprintf("(")

		if r.PlatformRegion != "" {
			ruleStr += fmt.Sprintf("PR=%s", r.PlatformRegion)

			if r.HyperscalerRegion != "" {
				ruleStr += fmt.Sprintf(", ")
			}
		}

		if r.HyperscalerRegion != "" {
			ruleStr += fmt.Sprintf("HR=%s", r.HyperscalerRegion)
		}

		ruleStr += fmt.Sprintf(")")
	}

	if r.EuAccess || r.Shared {
		ruleStr += fmt.Sprintf("-> ")

		if r.EuAccess {
			ruleStr += fmt.Sprintf("EU")

			if r.Shared {
				ruleStr += fmt.Sprintf(", ")
			}
		}

		if r.Shared {
			ruleStr += fmt.Sprintf("Shared")
		}

        ruleStr += fmt.Sprintf(" %-15s#  ", "\t")

		labels := r.Labels()
		for _, label := range labels {
			ruleStr += fmt.Sprintf("%s, ", string(label))
		}
	}

	return ruleStr
}
