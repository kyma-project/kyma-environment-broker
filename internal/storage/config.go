package storage

import (
	"fmt"
	"time"
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
}

func (cfg *Config) ConnectionURL() string {
	baseURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	if cfg.SSLMode != "disable" && cfg.SSLRootCert != "" {
		return fmt.Sprintf("%s sslrootcert=%s", baseURL, cfg.SSLRootCert)
	}

	return baseURL
}
