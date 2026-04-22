package main

import (
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
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		slog.Error("failed to ping DB", "error", err)
		os.Exit(1)
	}

	reader := analytics.NewDBReader(conn.NewSession(nil))

	var (
		mu    sync.RWMutex
		cache analytics.StatsResponse
	)

	refresh := func() {
		resp, err := buildStats(reader)
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
		mu.RLock()
		data := cache
		mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode stats", "error", err)
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

func buildStats(reader *analytics.DBReader) (analytics.StatsResponse, error) {
	provParams, err := reader.FetchActiveProvisioningParams()
	if err != nil {
		return analytics.StatsResponse{}, err
	}
	updateParams, err := reader.FetchUpdateParams()
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

// indexHTMLReader is replaced with an embedded file in Task 5.
func indexHTMLReader() *strings.Reader {
	return strings.NewReader("<html><body>KEB Analytics — UI coming soon</body></html>")
}
