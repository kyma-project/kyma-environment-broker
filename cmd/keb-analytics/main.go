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

	"github.com/kyma-project/kyma-environment-broker/internal"
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
	provParams    []internal.ProvisioningParameters
	updateParams  []internal.UpdatingParametersDTO
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
		mu    sync.RWMutex
		c     cache
	)

	refresh := func() {
		provParams, err := reader.FetchActiveProvisioningParamsInRange(analytics.TimeRange{})
		if err != nil {
			slog.Error("failed to fetch provisioning params", "error", err)
			return
		}
		updateParams, err := reader.FetchUpdateParamsInRange(analytics.TimeRange{})
		if err != nil {
			slog.Error("failed to fetch update params", "error", err)
			return
		}

		plans, regionsByPlan := analytics.BuildPlanRegionIndex(provParams, planIDToName)
		resp := analytics.StatsResponse{
			TotalInstances: len(provParams),
			Provisioning:   analytics.AggregateProvisioning(provParams),
			Updates:        analytics.AggregateUpdates(updateParams),
			Distributions:  analytics.BuildDistributions(provParams),
			Plans:          plans,
			RegionsByPlan:  regionsByPlan,
		}

		mu.Lock()
		c = cache{
			resp:          resp,
			provParams:    provParams,
			updateParams:  updateParams,
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
				data = buildFilteredStats(snapshot.provParams, snapshot.updateParams, planFilter, regionFilter, planIDToName, snapshot.plans, snapshot.regionsByPlan)
			}
		} else {
			// Time-range query: fetch from DB, then filter.
			provParams, err := reader.FetchActiveProvisioningParamsInRange(tr)
			if err != nil {
				slog.Error("failed to fetch provisioning params for range", "error", err)
				http.Error(w, "failed to build stats", http.StatusInternalServerError)
				return
			}
			updateParams, err := reader.FetchUpdateParamsInRange(tr)
			if err != nil {
				slog.Error("failed to fetch update params for range", "error", err)
				http.Error(w, "failed to build stats", http.StatusInternalServerError)
				return
			}
			plans, regionsByPlan := analytics.BuildPlanRegionIndex(provParams, planIDToName)
			data = buildFilteredStats(provParams, updateParams, planFilter, regionFilter, planIDToName, plans, regionsByPlan)
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
func buildFilteredStats(
	provParams []internal.ProvisioningParameters,
	updateParams []internal.UpdatingParametersDTO,
	planFilter, regionFilter string,
	planIDToName map[string]string,
	plans []string,
	regionsByPlan map[string][]string,
) analytics.StatsResponse {
	filtered := provParams
	if planFilter != "" {
		filtered = analytics.FilterByPlan(filtered, planFilter, planIDToName)
	}
	if regionFilter != "" {
		filtered = analytics.FilterByRegion(filtered, regionFilter)
	}
	return analytics.StatsResponse{
		TotalInstances: len(filtered),
		Provisioning:   analytics.AggregateProvisioning(filtered),
		Updates:        analytics.AggregateUpdates(updateParams),
		Distributions:  analytics.BuildDistributions(filtered),
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
