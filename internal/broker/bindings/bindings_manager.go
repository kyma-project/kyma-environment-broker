package broker

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	authv1 "k8s.io/api/authentication/v1"
	mv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Credentials struct {
}

type BindingsManager interface {
	Create(ctx context.Context, instance *internal.Instance, bindingID string) (string, error)
}

type ClientProvider interface {
	K8sClientSetForRuntimeID(runtimeID string) (*kubernetes.Clientset, error)
}

type KubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeId string) ([]byte, error)
}

type TokenRequestBindingsManager struct {
	clientProvider         ClientProvider
	tokenExpirationSeconds int
	kubeconfigBuilder      *kubeconfig.Builder
}

func NewTokenRequestBindingsManager(clientProvider ClientProvider, kubeconfigProvider KubeconfigProvider, tokenExpirationSeconds int) *TokenRequestBindingsManager {
	return &TokenRequestBindingsManager{
		clientProvider:         clientProvider,
		tokenExpirationSeconds: tokenExpirationSeconds,
		kubeconfigBuilder:      kubeconfig.NewBuilder(nil, nil, kubeconfigProvider),
	}
}

func (c *TokenRequestBindingsManager) Create(ctx context.Context, instance *internal.Instance, bindingID string) (string, error) {
	clientset, err := c.clientProvider.K8sClientSetForRuntimeID(instance.RuntimeID)

	if err != nil {
		return "", fmt.Errorf("while creating a runtime client for binding creation: %v", err)
	}

	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: mv1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: ptr.Integer64(int64(c.tokenExpirationSeconds)),
		},
	}

	// old usage with client.Client
	tkn, err := clientset.CoreV1().ServiceAccounts("default").CreateToken(ctx, "default", tokenRequest, mv1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("while creating a token request: %v", err)
	}

	kubeconfigContent, err := c.kubeconfigBuilder.BuildFromAdminKubeconfigForBinding(instance.RuntimeID, tkn.Status.Token)

	if err != nil {
		return "", fmt.Errorf("while creating a kubeconfig: %v", err)
	}

	return string(kubeconfigContent), nil
}
