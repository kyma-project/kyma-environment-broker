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
			expected: "host=localhost port=5432 user=postgres password=password dbname=broker sslmode=disable",
		},
		{
			name: "with SSL enabled",
			config: Config{
				User:        "user",
				Password:    "pass",
				Host:        "db.example.com",
				Port:        "5432",
				Name:        "mydb",
				SSLMode:     "require",
				SSLRootCert: "/path/to/cert",
			},
			expected: "host=db.example.com port=5432 user=user password=pass dbname=mydb sslmode=require sslrootcert=/path/to/cert",
		},
		{
			name: "with timezone set",
			config: Config{
				User:        "user",
				Password:    "pass",
				Host:        "db.example.com",
				Port:        "5432",
				Name:        "mydb",
				SSLMode:     "require",
				SSLRootCert: "/path/to/cert",
				Timezone:    "UTC",
			},
			expected: "host=db.example.com port=5432 user=user password=pass dbname=mydb sslmode=require sslrootcert=/path/to/cert timezone=UTC",
		},
		{
			name: "with empty timezone",
			config: Config{
				User:        "user",
				Password:    "pass",
				Host:        "db.example.com",
				Port:        "5432",
				Name:        "mydb",
				SSLMode:     "require",
				SSLRootCert: "/path/to/cert",
				Timezone:    "",
			},
			expected: "host=db.example.com port=5432 user=user password=pass dbname=mydb sslmode=require sslrootcert=/path/to/cert",
		},
		{
			name: "with timezone set",
			config: Config{
				User:        "user",
				Password:    "pass",
				Host:        "db.example.com",
				Port:        "5432",
				Name:        "mydb",
				SSLMode:     "disable",
				SSLRootCert: "/path/to/cert",
				Timezone:    "UTC",
			},
			expected: "host=db.example.com port=5432 user=user password=pass dbname=mydb sslmode=disable timezone=UTC",
		},
		{
			name: "with empty timezone",
			config: Config{
				User:        "user",
				Password:    "pass",
				Host:        "db.example.com",
				Port:        "5432",
				Name:        "mydb",
				SSLMode:     "disable",
				SSLRootCert: "/path/to/cert",
				Timezone:    "",
			},
			expected: "host=db.example.com port=5432 user=user password=pass dbname=mydb sslmode=disable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ConnectionURL()
			assert.Equal(t, tt.expected, got)
		})
	}
}
