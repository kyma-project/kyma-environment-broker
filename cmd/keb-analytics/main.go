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

// rangeCache holds pre-computed stats for a single time window.
type rangeCache struct {
	provParams   []analytics.ProvisioningParamsWithID
	updateParams []analytics.UpdateParamsWithID
	resp         analytics.StatsResponse // unfiltered stats for this window
}

// cache is the top-level in-memory store populated on each refresh.
type cache struct {
	opEvents      []analytics.OpEvent   // full history, shared across all windows
	byRange       map[string]rangeCache // keys: "all", "7d", "30d", "90d"
	plans         []string              // from the "all" window
	regionsByPlan map[string][]string   // from the "all" window
	cachedAt      time.Time
	nextRefreshAt time.Time
}

// buildRangeCache computes provParams, updateParams and the unfiltered StatsResponse
// for the given time window from the shared opEvents slice.
// Trends are intentionally built from the full opEvents history regardless of the window:
// this means the Trends field in the 7d/30d/90d responses is identical to the "all" response,
// providing continuous historical context on the trend chart even when a narrow period is selected.
func buildRangeCache(opEvents []analytics.OpEvent, tr analytics.TimeRange, planIDToName map[string]string, trendParams []string) rangeCache {
	provParams := analytics.OpEventsToProvParamsInRange(opEvents, tr)
	updateParams := analytics.OpEventsToUpdateParamsInRange(opEvents, tr)
	plans, regionsByPlan := analytics.BuildPlanRegionIndex(provParams, planIDToName)

	// Run independent aggregations in parallel.
	var (
		provisioning  analytics.ParameterStats
		updates       analytics.ParameterStats
		combined      analytics.ParameterStats
		distributions []analytics.DistributionStat
	)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); provisioning = analytics.AggregateProvisioning(provParams) }()
	go func() { defer wg.Done(); updates = analytics.AggregateUpdates(provParams, updateParams) }()
	go func() { defer wg.Done(); combined = analytics.AggregateCombined(provParams, updateParams) }()
	go func() { defer wg.Done(); distributions = analytics.BuildDistributions(provParams) }()
	wg.Wait()

	trends := analytics.BuildTrends(opEvents, trendParams)
	resp := analytics.StatsResponse{
		TotalInstances: len(provParams),
		TotalUpdates:   len(updateParams),
		Provisioning:   provisioning,
		Updates:        updates,
		Combined:       combined,
		Distributions:  distributions,
		Trends:         trends,
		Plans:          plans,
		RegionsByPlan:  regionsByPlan,
	}
	return rangeCache{provParams: provParams, updateParams: updateParams, resp: resp}
}

// matchRangeKey returns the cache key ("7d", "30d", "90d", "all") if the TimeRange
// matches one of the pre-computed windows (within 1-day tolerance for client clock skew).
// Returns "" if no match — caller must slice from opEvents in-memory.
func matchRangeKey(tr analytics.TimeRange) string {
	if tr.From.IsZero() && tr.To.IsZero() {
		return "all"
	}
	if !tr.To.IsZero() {
		return "" // bounded "to" not a standard window
	}
	age := time.Since(tr.From)
	const tolerance = 25 * time.Hour
	switch {
	case abs(age-7*24*time.Hour) < tolerance:
		return "7d"
	case abs(age-30*24*time.Hour) < tolerance:
		return "30d"
	case abs(age-90*24*time.Hour) < tolerance:
		return "90d"
	}
	return ""
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
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
		mu         sync.RWMutex
		c          cache
		refreshing bool
	)

	var refresh func()

	// runRefresh executes refresh() under a refreshing flag, safe for concurrent callers.
	// A second call while one is already in progress is a no-op.
	var refreshMu sync.Mutex
	runRefresh := func() {
		if !refreshMu.TryLock() {
			return // already in progress
		}
		defer refreshMu.Unlock()

		mu.Lock()
		refreshing = true
		mu.Unlock()

		refresh()

		mu.Lock()
		refreshing = false
		mu.Unlock()
	}

	refresh = func() {
		opEvents, err := reader.FetchOpEventsInRange(analytics.TimeRange{})
		if err != nil {
			slog.Error("failed to fetch op events", "error", err)
			return
		}

		now := time.Now().UTC()
		windows := map[string]analytics.TimeRange{
			"all": {},
			"7d":  {From: now.AddDate(0, 0, -7)},
			"30d": {From: now.AddDate(0, 0, -30)},
			"90d": {From: now.AddDate(0, 0, -90)},
		}

		// Build the "all" window first to derive trendParams used by all windows.
		allRC := buildRangeCache(opEvents, analytics.TimeRange{}, planIDToName, nil)
		trendParams := analytics.TrendParamsFrom(allRC.resp.Combined)

		// Rebuild "all" with trendParams so its Trends field is populated.
		allRC = buildRangeCache(opEvents, analytics.TimeRange{}, planIDToName, trendParams)

		byRange := make(map[string]rangeCache, len(windows))
		byRange["all"] = allRC
		for key, tr := range windows {
			if key == "all" {
				continue
			}
			byRange[key] = buildRangeCache(opEvents, tr, planIDToName, trendParams)
		}

		plans, regionsByPlan := allRC.resp.Plans, allRC.resp.RegionsByPlan

		mu.Lock()
		c = cache{
			opEvents:      opEvents,
			byRange:       byRange,
			plans:         plans,
			regionsByPlan: regionsByPlan,
			cachedAt:      now,
			nextRefreshAt: now.Add(cfg.RefreshInterval),
		}
		mu.Unlock()
		slog.Info("stats cache refreshed", "total_instances", allRC.resp.TotalInstances)
	}

	runRefresh()
	go func() {
		ticker := time.NewTicker(cfg.RefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			runRefresh()
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

		mu.RLock()
		snapshot := c
		mu.RUnlock()

		if snapshot.byRange == nil {
			http.Error(w, "data not yet available, try again shortly", http.StatusServiceUnavailable)
			return
		}

		// Resolve the rangeCache to use — pre-computed if the window matches, otherwise
		// slice from the full opEvents in-memory (no DB query in either path).
		var rc rangeCache
		if key := matchRangeKey(tr); key != "" {
			rc = snapshot.byRange[key]
		} else {
			// Custom date range: derive in-memory from cached opEvents.
			allCombined := snapshot.byRange["all"].resp.Combined
			trendParams := analytics.TrendParamsFrom(allCombined)
			rc = buildRangeCache(snapshot.opEvents, tr, planIDToName, trendParams)
		}

		var data analytics.StatsResponse
		if planFilter == "" && regionFilter == "" {
			data = rc.resp
			// Always use the full plan/region index for dropdowns.
			data.Plans = snapshot.plans
			data.RegionsByPlan = snapshot.regionsByPlan
		} else {
			allCombined := snapshot.byRange["all"].resp.Combined
			trendParams := analytics.TrendParamsFrom(allCombined)
			data = buildFilteredStats(rc.provParams, rc.updateParams, snapshot.opEvents, planFilter, regionFilter, planIDToName, snapshot.plans, snapshot.regionsByPlan, trendParams)
		}
		data.CachedAt = snapshot.cachedAt.Format(time.RFC3339)
		data.NextRefreshAt = snapshot.nextRefreshAt.Format(time.RFC3339)

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
		go runRefresh()
		w.WriteHeader(http.StatusAccepted)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		mu.RLock()
		isRefreshing := refreshing
		cachedAt := c.cachedAt
		nextRefreshAt := c.nextRefreshAt
		mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"refreshing":      isRefreshing,
			"cached_at":       cachedAt.Format(time.RFC3339),
			"next_refresh_at": nextRefreshAt.Format(time.RFC3339),
		}); err != nil {
			slog.Error("failed to encode status", "error", err)
		}
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
		Updates:        analytics.AggregateUpdates(filtered, updateParams),
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
