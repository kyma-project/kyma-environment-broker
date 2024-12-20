package test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/testing/e2e/provisioning/pkg/client/broker"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
)

type UpgradeConfig struct {
	UpgradeTimeout time.Duration `default:"3h"`
}

type UpgradeSuite struct {
	upgradeClient *broker.UpgradeClient

	UpgradeTimeout time.Duration
}

func newUpgradeSuite(t *testing.T, ctx context.Context, oAuthConfig broker.BrokerOAuthConfig, config broker.Config, log *slog.Logger) *UpgradeSuite {
	cfg := &UpgradeConfig{}
	err := envconfig.InitWithPrefix(cfg, "APP")
	require.NoError(t, err)

	upgradeClient := broker.NewUpgradeClient(ctx, oAuthConfig, config, log)

	return &UpgradeSuite{
		upgradeClient:  upgradeClient,
		UpgradeTimeout: cfg.UpgradeTimeout,
	}
}
