package broker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestCredentialsManagerImpl_Create(t *testing.T) {
	t.SkipNow()
	t.Run("should create a new service binding without error", func(t *testing.T) {
		k8sCfg, err := config.GetConfig()
		if err != nil {
			t.Errorf("unable to get k8s config: %v", err)
			return
		}

		cli, err := initClient(k8sCfg)
		if err != nil {
			t.Errorf("unable to get k8s config: %v", err)
			return
		}

		skrK8sClientProvider := kubeconfig.NewK8sClientFromSecretProvider(cli)

		manager := &TokenRequestsBindingsManager{tokenExpiration: 600, clientProvider: skrK8sClientProvider, kubeconfigBuilder: kubeconfig.NewBuilder(nil, nil, skrK8sClientProvider)}

		ctx := context.Background()
		runtimeID := "fdf57b6f-5531-485d-9af7-7baa7a7cac01"
		bindingID := "2"
		// details := domain.BindDetails{
		//     ServiceID: "test-service-id",
		//     PlanID:    "test-plan-id",
		// }

		// Act
		kubeconfig, err := manager.Create(ctx, runtimeID, bindingID)

		// Assert
		require.NoError(t, err)
		fmt.Printf("Kubeconfig is: %s", kubeconfig)
	})
}

func initClient(cfg *rest.Config) (client.Client, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("while waiting for client mapper: %w", err)
		}
	}
	cli, err := client.New(cfg, client.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("while creating a client: %w", err)
	}
	return cli, nil
}
