package broker

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	mv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Credentials struct {
}

type BindingsManager interface {
	Create(ctx context.Context, instance *internal.Instance, bindingID string, expirationSeconds int) (string, error)
}

type ClientProvider interface {
	K8sClientSetForRuntimeID(runtimeID string) (*kubernetes.Clientset, error)
}

type KubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeId string) ([]byte, error)
}

type TokenRequestBindingsManager struct {
	clientProvider    ClientProvider
	kubeconfigBuilder *kubeconfig.Builder
}

func NewTokenRequestBindingsManager(clientProvider ClientProvider, kubeconfigProvider KubeconfigProvider) *TokenRequestBindingsManager {
	return &TokenRequestBindingsManager{
		clientProvider:    clientProvider,
		kubeconfigBuilder: kubeconfig.NewBuilder(nil, nil, kubeconfigProvider),
	}
}

func (c *TokenRequestBindingsManager) Create(ctx context.Context, instance *internal.Instance, bindingID string, expirationSeconds int) (string, error) {
	clientset, err := c.clientProvider.K8sClientSetForRuntimeID(instance.RuntimeID)

	if err != nil {
		return "", fmt.Errorf("while creating a runtime client for binding creation: %v", err)
	}

	serviceBindingName := fmt.Sprintf("kyma-binding-%s", bindingID)

	_, err = clientset.CoreV1().ServiceAccounts("kyma-system").Create(ctx,
		&v1.ServiceAccount{
			ObjectMeta: mv1.ObjectMeta{
				Name:      serviceBindingName,
				Namespace: "kyma-system",
				Labels:    map[string]string{"app.kubernetes.io/managed-by": "kcp-kyma-environment-broker"},
			},
		}, mv1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("while creating a service account: %v", err)
	}

	_, err = clientset.RbacV1().ClusterRoles().Create(ctx,
		&rbacv1.ClusterRole{
			TypeMeta: mv1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "ClusterRole"},
			ObjectMeta: mv1.ObjectMeta{
				Name:   serviceBindingName,
				Labels: map[string]string{"app.kubernetes.io/managed-by": "kcp-kyma-environment-broker"},
			},
			Rules: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
		}, mv1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("while creating a cluster role: %v", err)
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
		TypeMeta: mv1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "ClusterRoleBinding"},
		ObjectMeta: mv1.ObjectMeta{
			Name:   serviceBindingName,
			Labels: map[string]string{"app.kubernetes.io/managed-by": "kcp-kyma-environment-broker"},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     serviceBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: "kyma-system",
				Name:      serviceBindingName,
			},
		},
	}, mv1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("while creating a cluster role binding: %v", err)
	}

	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: mv1.ObjectMeta{
			Name:      serviceBindingName,
			Namespace: "kyma-system",
			Labels:    map[string]string{"app.kubernetes.io/managed-by": "kcp-kyma-environment-broker"},
		},
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: ptr.Integer64(int64(expirationSeconds)),
		},
	}

	tkn, err := clientset.CoreV1().ServiceAccounts("kyma-system").CreateToken(ctx, serviceBindingName, tokenRequest, mv1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("while creating a token request: %v", err)
	}

	kubeconfigContent, err := c.kubeconfigBuilder.BuildFromAdminKubeconfigForBinding(instance.RuntimeID, tkn.Status.Token)

	if err != nil {
		return "", fmt.Errorf("while creating a kubeconfig: %v", err)
	}

	return string(kubeconfigContent), nil
}
