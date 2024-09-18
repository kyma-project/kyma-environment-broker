package broker

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	authv1 "k8s.io/api/authentication/v1"
	mv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Credentials struct {
}

type BindingsManager interface {
	Create(ctx context.Context, runtimeID, bindingID string) (string, error)
}

type ClientProvider interface {
	K8sClientSetForRuntimeID(runtimeID string) (*kubernetes.Clientset, error)
}

type KubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeId string) ([]byte, error)
}

type TokenRequestsBindingsManager struct {
	clientProvider    ClientProvider
	tokenExpiration   int
	kubeconfigBuilder *kubeconfig.Builder
}

func NewTokenRequestsBindingsManager(clientProvider ClientProvider, kubeconfigProvider KubeconfigProvider, tokenExpiration int) *TokenRequestsBindingsManager {
	return &TokenRequestsBindingsManager{
		clientProvider:    clientProvider,
		tokenExpiration:   tokenExpiration,
		kubeconfigBuilder: kubeconfig.NewBuilder(nil, nil, kubeconfigProvider),
	}
}

func (c *TokenRequestsBindingsManager) Create(ctx context.Context, runtimeID, bindingID string) (string, error) {
	clientset, err := c.clientProvider.K8sClientSetForRuntimeID(runtimeID)

	if err != nil {
		return "", fmt.Errorf("while creating a runtime client for binding creation: %v", err)
	}

	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: mv1.ObjectMeta{
			Name:      "admin",
			Namespace: "default",
		},
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: ptr.Integer64(int64(c.tokenExpiration)),
		},
	}

	// old usage with client.Client
	tkn, err := clientset.CoreV1().ServiceAccounts("default").CreateToken(ctx, "admin", tokenRequest, mv1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("while creating a token request: %v", err)
	}

	kubeconfigContent, err := c.kubeconfigBuilder.BuildFromAdminKubeconfigForBinding(runtimeID, tkn.Status.Token)

	if err != nil {
		return "", fmt.Errorf("while creating a kubeconfig: %v", err)
	}

	return string(kubeconfigContent), nil
}
