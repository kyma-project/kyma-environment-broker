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

// DistributionStat holds value breakdown for a single parameter.
type DistributionStat struct {
	Parameter string         `json:"parameter"`
	Values    map[string]int `json:"values"`
}

// StatsResponse is the top-level JSON returned by GET /api/stats.
type StatsResponse struct {
	TotalInstances int                   `json:"total_instances"`
	Provisioning   ParameterStats        `json:"provisioning"`
	Updates        ParameterStats        `json:"updates"`
	Distributions  []DistributionStat    `json:"distributions"`
	Plans          []string              `json:"plans"`
	RegionsByPlan  map[string][]string   `json:"regions_by_plan"`
}
