package storage

import (
	"fmt"
	"time"
)

const (
	connectionURLFormat = "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s"
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
	Timezone        string        `envconfig:"optional"`
}

func (cfg *Config) ConnectionURL() string {
	url := fmt.Sprintf(connectionURLFormat, cfg.Host, cfg.Port, cfg.User,
		cfg.Password, cfg.Name, cfg.SSLMode, cfg.SSLRootCert)
	if cfg.Timezone != "" {
		url = fmt.Sprintf("%s timezone=%s", url, cfg.Timezone)
	}
	return url
}
