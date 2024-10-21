package provisioningservice

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	Provisioning ProvisioningConfig
}

type ProvisioningSuite struct {
	t      *testing.T
	logger *slog.Logger

	provisioningClient *ProvisioningClient
}

func NewProvisioningSuite(t *testing.T) *ProvisioningSuite {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx := context.Background()

	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	require.NoError(t, err)

	logger.Info("Creating a new provisioning client")
	provisioningClient := NewProvisioningClient(cfg.Provisioning, logger, ctx, 60)
	err = provisioningClient.GetAccessToken()
	require.NoError(t, err)

	return &ProvisioningSuite{
		t:                  t,
		logger:             logger,
		provisioningClient: provisioningClient,
	}
}
