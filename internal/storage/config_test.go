package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ConnectionURL(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "default config",
			config: Config{
				User:     "postgres",
				Password: "password",
				Host:     "localhost",
				Port:     "5432",
				Name:     "broker",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=broker sslmode=disable timezone=UTC",
		},
		{
			name: "with SSL enabled",
			config: Config{
				User:        "postgres",
				Password:    "password",
				Host:        "localhost",
				Port:        "5432",
				Name:        "broker",
				SSLMode:     "require",
				SSLRootCert: "/path/to/cert",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=broker sslmode=require timezone=UTC sslrootcert=/path/to/cert",
		},
		{
			name: "with SSL enabled",
			config: Config{
				User:        "postgres",
				Password:    "password",
				Host:        "localhost",
				Port:        "5432",
				Name:        "broker",
				SSLMode:     "verify-full",
				SSLRootCert: "/path/to/cert",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=broker sslmode=verify-full timezone=UTC sslrootcert=/path/to/cert",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ConnectionURL()
			assert.Equal(t, tt.expected, got)
		})
	}
}
