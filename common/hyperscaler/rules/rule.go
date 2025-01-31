package rules

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