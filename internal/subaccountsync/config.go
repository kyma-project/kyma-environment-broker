package subaccountsync

import (
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type (
	Config struct {
		CisAccounts                       CisEndpointConfig
		CisEvents                         CisEndpointConfig
		Database                          storage.Config
		UpdateResources                   bool          `envconfig:"default=false"`
		EventsWindowSize                  time.Duration `envconfig:"default=20m"`
		EventsWindowInterval              time.Duration `envconfig:"default=15m"`
		AccountsSyncInterval              time.Duration `envconfig:"default=24h"`
		StorageSyncInterval               time.Duration `envconfig:"default=10m"`
		SyncQueueSleepInterval            time.Duration `envconfig:"default=30s"`
		MetricsPort                       string        `envconfig:"default=8081"`
		LogLevel                          string        `envconfig:"default=info"`
		RuntimeConfigurationConfigMapName string
		AlwaysSubaccountFromDatabase      bool `envconfig:"default=false"`
	}

	CisEndpointConfig struct {
		ClientID               string
		ClientSecret           string
		AuthURL                string
		ServiceURL             string
		PageSize               string        `envconfig:"default=150,optional"`
		RateLimitingInterval   time.Duration `envconfig:"default=2s,optional"`
		MaxRequestsPerInterval int           `envconfig:"default=5,optional"`
	}
)

func (c Config) GetLogLevel() slog.Level {
	switch strings.ToUpper(c.LogLevel) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
