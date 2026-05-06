package analytics

import (
	"encoding/json"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

// NewStatsHandler returns an http.HandlerFunc that queries the DB directly on
// every request and returns a StatsResponse. Intended for test use only.
func NewStatsHandler(reader *DBReader) http.HandlerFunc {
	planIDToName := make(map[string]string, len(broker.PlanIDsMapping))
	for name, id := range broker.PlanIDsMapping {
		planIDToName[string(id)] = string(name)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		provParams, err := reader.FetchActiveProvisioningParams()
		if err != nil {
			http.Error(w, "failed to fetch provisioning params", http.StatusInternalServerError)
			return
		}
		updateParams, err := reader.FetchUpdateParams()
		if err != nil {
			http.Error(w, "failed to fetch update params", http.StatusInternalServerError)
			return
		}

		plans, regionsByPlan := BuildPlanRegionIndex(provParams, planIDToName)
		resp := StatsResponse{
			TotalInstances: len(provParams),
			Provisioning:   AggregateProvisioning(provParams),
			Updates:        AggregateUpdates(updateParams),
			Distributions:  BuildDistributions(provParams),
			Plans:          plans,
			RegionsByPlan:  regionsByPlan,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}
