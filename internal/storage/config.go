package storage

import (
	"fmt"
	"time"
)

const (
	connectionURLFormat = "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=UTC"
)

type Config struct {
	User        string `envconfig:"default=postgres"`
	Password    string `envconfig:"default=password"`
	Host        string `envconfig:"default=localhost"`
	Port        string `envconfig:"default=5432"`
	Name        string `envconfig:"default=broker"`
	SSLMode     string `envconfig:"default=disable"`
	SSLRootCert string `envconfig:"optional"`

	SecretKey string `envconfig:"optional"`

	MaxOpenConns    int           `envconfig:"default=8"`
	MaxIdleConns    int           `envconfig:"default=2"`
	ConnMaxLifetime time.Duration `envconfig:"default=30m"`

	Fips FipsConfig
}

type FipsConfig struct {
	WriteGcm         bool          `envconfig:"default=false"`
	RewriteCfb       bool          `envconfig:"default=false"`
	RewriteBatchSize int           `envconfig:"default=100"`
	BatchInterval    time.Duration `envconfig:"default=1m"`
}

func (cfg *Config) ConnectionURL() string {
	url := fmt.Sprintf(connectionURLFormat, cfg.Host, cfg.Port, cfg.User,
		cfg.Password, cfg.Name, cfg.SSLMode)
	if cfg.SSLMode != "disable" && cfg.SSLMode != "" {
		url += fmt.Sprintf(" sslrootcert=%s", cfg.SSLRootCert)
	}
	return url
}
