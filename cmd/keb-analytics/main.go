package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocraft/dbr"
	_ "github.com/lib/pq"
	"github.com/vrischmann/envconfig"

	"github.com/kyma-project/kyma-environment-broker/internal/analytics"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

//go:embed static/index.html
var indexHTML string

type Config struct {
	Database struct {
		User     string `envconfig:"default=postgres"`
		Password string `envconfig:"default=password"`
		Host     string `envconfig:"default=localhost"`
		Port     string `envconfig:"default=5432"`
		Name     string `envconfig:"default=broker"`
		SSLMode  string `envconfig:"default=disable"`
	}
	Port            string        `envconfig:"default=8080"`
	RefreshInterval time.Duration `envconfig:"default=1h"`
}

type cache struct {
	resp          analytics.StatsResponse
	provParams    []analytics.ProvisioningParamsWithID
	updateParams  []analytics.UpdateParamsWithID
	opEvents      []analytics.OpEvent
	plans         []string
	regionsByPlan map[string][]string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	var cfg Config
	if err := envconfig.InitWithPrefix(&cfg, "APP"); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	connURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=UTC",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)
	conn, err := dbr.Open("postgres", connURL, nil)
	if err != nil {
		slog.Error("failed to open DB connection", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close DB connection", "error", err)
		}
	}()

	for {
		if err := conn.Ping(); err != nil {
			slog.Warn("DB not ready, retrying in 5s", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	reader := analytics.NewDBReader(conn.NewSession(nil))

	// Build planID → planName lookup from broker constants.
	planIDToName := make(map[string]string, len(broker.PlanIDsMapping))
	for name, id := range broker.PlanIDsMapping {
		planIDToName[string(id)] = string(name)
	}

	var (
		mu sync.RWMutex
		c  cache
	)

	refresh := func() {
		// Single query: fetch all op events (provision + update) for active instances.
		// provParams and updateParams are derived in-memory, eliminating two extra round-trips.
		opEvents, err := reader.FetchOpEventsInRange(analytics.TimeRange{})
		if err != nil {
			slog.Error("failed to fetch op events", "error", err)
			return
		}
		provParams := analytics.OpEventsToProvParamsInRange(opEvents, analytics.TimeRange{})
		updateParams := analytics.OpEventsToUpdateParamsInRange(opEvents, analytics.TimeRange{})

		plans, regionsByPlan := analytics.BuildPlanRegionIndex(provParams, planIDToName)
		provisioning := analytics.AggregateProvisioning(provParams)
		updates := analytics.AggregateUpdates(updateParams)
		combined := analytics.AggregateCombined(provParams, updateParams)
		trendParams := analytics.TrendParamsFrom(combined)
		trends := analytics.BuildTrends(opEvents, trendParams)
		resp := analytics.StatsResponse{
			TotalInstances: len(provParams),
			TotalUpdates:   len(updateParams),
			Provisioning:   provisioning,
			Updates:        updates,
			Combined:       combined,
			Distributions:  analytics.BuildDistributions(provParams),
			Trends:         trends,
			Plans:          plans,
			RegionsByPlan:  regionsByPlan,
		}

		mu.Lock()
		c = cache{
			resp:          resp,
			provParams:    provParams,
			updateParams:  updateParams,
			opEvents:      opEvents,
			plans:         plans,
			regionsByPlan: regionsByPlan,
		}
		mu.Unlock()
		slog.Info("stats cache refreshed", "total_instances", resp.TotalInstances)
	}

	refresh()
	go func() {
		ticker := time.NewTicker(cfg.RefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			refresh()
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		tr, err := parseTimeRange(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		planFilter := r.URL.Query().Get("plan")
		regionFilter := r.URL.Query().Get("region")

		var data analytics.StatsResponse

		if tr.From.IsZero() && tr.To.IsZero() {
			// Use cached full dataset; filter in memory if needed.
			mu.RLock()
			snapshot := c
			mu.RUnlock()

			if planFilter == "" && regionFilter == "" {
				data = snapshot.resp
			} else {
				data = buildFilteredStats(snapshot.provParams, snapshot.updateParams, snapshot.opEvents, planFilter, regionFilter, planIDToName, snapshot.plans, snapshot.regionsByPlan, analytics.TrendParamsFrom(snapshot.resp.Combined))
			}
		} else {
			// Time-range query: slice the in-memory cache — no DB round-trip needed.
			mu.RLock()
			snapshot := c
			mu.RUnlock()
			provParams := analytics.OpEventsToProvParamsInRange(snapshot.opEvents, tr)
			updateParams := analytics.OpEventsToUpdateParamsInRange(snapshot.opEvents, tr)
			plans, regionsByPlan := analytics.BuildPlanRegionIndex(provParams, planIDToName)
			trendParams := analytics.TrendParamsFrom(snapshot.resp.Combined)
			data = buildFilteredStats(provParams, updateParams, snapshot.opEvents, planFilter, regionFilter, planIDToName, plans, regionsByPlan, trendParams)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode stats", "error", err)
		}
	})

	mux.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		refresh()
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "index.html", time.Time{}, indexHTMLReader())
	})

	addr := ":" + cfg.Port
	slog.Info("starting keb-analytics server", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// buildFilteredStats filters provParams/updateParams by plan and region, then aggregates.
// plans and regionsByPlan are always the full unfiltered index (for dropdown population).
// opEvents are unfiltered (trends are not affected by plan/region filter).
// trendParams is the list of parameter names to build trends for; it must come from the
// full (unfiltered) combined stats so that trends remain populated even when the selected
// time-range window contains no provisioning operations.
func buildFilteredStats(
	provParams []analytics.ProvisioningParamsWithID,
	updateParams []analytics.UpdateParamsWithID,
	opEvents []analytics.OpEvent,
	planFilter, regionFilter string,
	planIDToName map[string]string,
	plans []string,
	regionsByPlan map[string][]string,
	trendParams []string,
) analytics.StatsResponse {
	filtered := provParams
	if planFilter != "" {
		filtered = analytics.FilterByPlan(filtered, planFilter, planIDToName)
	}
	if regionFilter != "" {
		filtered = analytics.FilterByRegion(filtered, regionFilter)
	}
	combined := analytics.AggregateCombined(filtered, updateParams)
	trends := analytics.BuildTrends(opEvents, trendParams)
	return analytics.StatsResponse{
		TotalInstances: len(filtered),
		TotalUpdates:   len(updateParams),
		Provisioning:   analytics.AggregateProvisioning(filtered),
		Updates:        analytics.AggregateUpdates(updateParams),
		Combined:       combined,
		Distributions:  analytics.BuildDistributions(filtered),
		Trends:         trends,
		Plans:          plans,
		RegionsByPlan:  regionsByPlan,
	}
}

// parseTimeRange reads optional ?from=YYYY-MM-DD and ?to=YYYY-MM-DD query params.
func parseTimeRange(r *http.Request) (analytics.TimeRange, error) {
	var tr analytics.TimeRange
	if s := r.URL.Query().Get("from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return tr, fmt.Errorf("invalid 'from' date %q, expected YYYY-MM-DD", s)
		}
		tr.From = t.UTC()
	}
	if s := r.URL.Query().Get("to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return tr, fmt.Errorf("invalid 'to' date %q, expected YYYY-MM-DD", s)
		}
		tr.To = t.UTC()
	}
	return tr, nil
}

func indexHTMLReader() *strings.Reader {
	return strings.NewReader(indexHTML)
}
