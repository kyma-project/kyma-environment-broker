package analytics

// ParameterStat holds usage count for a single parameter value.
type ParameterStat struct {
	Parameter string `json:"parameter"`
	SetCount  int    `json:"set_count"`
	Total     int    `json:"total"`
}

// ParameterStats is a ranked list of parameter usage.
type ParameterStats struct {
	Parameters []ParameterStat `json:"parameters"`
}

// CountFor returns the SetCount for the named parameter, or 0 if not present.
func (ps ParameterStats) CountFor(name string) int {
	for _, p := range ps.Parameters {
		if p.Parameter == name {
			return p.SetCount
		}
	}
	return 0
}

// DistributionStat holds value breakdown for a single parameter.
type DistributionStat struct {
	Parameter string         `json:"parameter"`
	Values    map[string]int `json:"values"`
}

// TrendPoint holds the count of instances with a parameter set on a given day.
type TrendPoint struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int    `json:"count"` // cumulative count of active instances with param set
	Total int    `json:"total"` // cumulative count of active instances provisioned by this day
}

// TrendStat holds the daily trend for a single parameter.
type TrendStat struct {
	Parameter string       `json:"parameter"`
	Points    []TrendPoint `json:"points"`
}

// StatsResponse is the top-level JSON returned by GET /api/stats.
type StatsResponse struct {
	TotalInstances int                 `json:"total_instances"`
	TotalUpdates   int                 `json:"total_updates"`
	Provisioning   ParameterStats      `json:"provisioning"`
	Updates        ParameterStats      `json:"updates"`
	Combined       ParameterStats      `json:"combined"`
	Distributions  []DistributionStat  `json:"distributions"`
	Trends         []TrendStat         `json:"trends"`
	Plans          []string            `json:"plans"`
	RegionsByPlan  map[string][]string `json:"regions_by_plan"`
}
