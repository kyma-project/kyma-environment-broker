package broker

import (
	"context"
	"fmt"

	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	shoot "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GardenerBindingManager struct {
	gardenerClient client.Client
}

func NewGardenerBindingManager(gardenerClient client.Client) *GardenerBindingManager {
	return &GardenerBindingManager{
		gardenerClient: gardenerClient,
	}
}

func (c *GardenerBindingManager) Create(ctx context.Context, instance *internal.Instance, bindingID string, expirationSeconds int) (string, error) {

	shoot := &shoot.Shoot{
		TypeMeta: metav1.TypeMeta{APIVersion: "core.gardener.cloud/v1beta1", Kind: "Shoot"},
	}
	err := c.gardenerClient.Get(context.Background(), client.ObjectKey{Name: instance.InstanceDetails.ShootName, Namespace: "garden-kyma-dev"}, shoot)
	if err != nil {
		return "", fmt.Errorf("while getting shoot: %v", err)
	}

	adminKubeconfigRequest := &authenticationv1alpha1.AdminKubeconfigRequest{
		Spec: authenticationv1alpha1.AdminKubeconfigRequestSpec{
			ExpirationSeconds: ptr.Integer64(int64(expirationSeconds)),
		},
	}

	err = c.gardenerClient.SubResource("adminkubeconfig").Create(context.Background(), shoot, adminKubeconfigRequest)
	if err != nil {
		return "", fmt.Errorf("while creating admin kubeconfig request: %v", err)
	}
	shootKubeconfig := adminKubeconfigRequest.Status.Kubeconfig

	return string(shootKubeconfig), nil
}
