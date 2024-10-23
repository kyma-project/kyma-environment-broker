package cmd

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/testing/e2e/skr/provisioning-service-test/internal"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	secretNamespace = "kyma-system"
	secretName      = "sap-btp-manager"
)

type Config struct {
	Provisioning internal.ProvisioningConfig
}

type ProvisioningSuite struct {
	t      *testing.T
	logger *slog.Logger

	provisioningClient *internal.ProvisioningClient
}

func NewProvisioningSuite(t *testing.T) *ProvisioningSuite {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx := context.Background()

	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	require.NoError(t, err)

	logger.Info("Creating a new provisioning client")
	provisioningClient := internal.NewProvisioningClient(cfg.Provisioning, logger, ctx, 60)
	err = provisioningClient.GetAccessToken()
	require.NoError(t, err)

	return &ProvisioningSuite{
		t:                  t,
		logger:             logger,
		provisioningClient: provisioningClient,
	}
}

func (p *ProvisioningSuite) K8sClientSetForKubeconfig(kubeconfig string) (kubernetes.Interface, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
