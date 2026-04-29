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

	var (
		mu    sync.RWMutex
		cache analytics.StatsResponse
	)

	refresh := func() {
		resp, err := buildStats(reader, analytics.TimeRange{})
		if err != nil {
			slog.Error("failed to build stats", "error", err)
			return
		}
		mu.Lock()
		cache = resp
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
		var data analytics.StatsResponse
		if tr.From.IsZero() && tr.To.IsZero() {
			mu.RLock()
			data = cache
			mu.RUnlock()
		} else {
			data, err = buildStats(reader, tr)
			if err != nil {
				slog.Error("failed to build stats for range", "error", err)
				http.Error(w, "failed to build stats", http.StatusInternalServerError)
				return
			}
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

func buildStats(reader *analytics.DBReader, tr analytics.TimeRange) (analytics.StatsResponse, error) {
	provParams, err := reader.FetchActiveProvisioningParamsInRange(tr)
	if err != nil {
		return analytics.StatsResponse{}, err
	}
	updateParams, err := reader.FetchUpdateParamsInRange(tr)
	if err != nil {
		return analytics.StatsResponse{}, err
	}
	return analytics.StatsResponse{
		TotalInstances: len(provParams),
		Provisioning:   analytics.AggregateProvisioning(provParams),
		Updates:        analytics.AggregateUpdates(updateParams),
		Distributions:  analytics.BuildDistributions(provParams),
	}, nil
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
