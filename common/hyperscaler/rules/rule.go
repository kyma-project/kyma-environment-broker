package rules

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Rule struct { 
    Plan string
    PlatformRegion string
    HyperscalerRegion string
    EuAccess bool
    Shared bool
}

func (r *Rule) Labels() string {
    if "azure" == r.Plan {
        return "hyperscalerType: azure"
    } 
    return ""
}

func (r *Rule) Matched() bool {
    if "azure" == r.Plan {
        return true
    } 
    
    return false
}


func (r* Rule) SetAttributeValue(attribute, value string) (*Rule, error) {
   switch attribute {
    case "PR":
        if r.PlatformRegion != "" {
            return nil, fmt.Errorf("PlatformRegion already set");
        } else if value == "" {
            return nil, fmt.Errorf("PlatformRegion is empty")
        }

        r.PlatformRegion = value
    case "HR":
        if r.HyperscalerRegion != "" {
            return nil, fmt.Errorf("HyperscalerRegion already set");
        } else if value == "" {
            return nil, fmt.Errorf("HyperscalerRegion is empty")
        }

        r.HyperscalerRegion = value
    case "EU":
        if r.EuAccess {
            return nil, fmt.Errorf("EuAccess already set");
        }
        r.EuAccess = true
    case "S":
        if r.Shared {
            return nil, fmt.Errorf("Shared already set");
        }

        r.Shared = true
    default:
        return nil, fmt.Errorf("unknown attribute %s", attribute)
    }

    return r, nil
}

func (r* Rule) SetPlan(value string) (*Rule, error) {
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

func (r* Rule) NumberOfInputAtributes() int {
    count:=0

    if r.PlatformRegion != "" {
        count++
    }

    if r.HyperscalerRegion != "" {
        count++
    }

    return count
}


