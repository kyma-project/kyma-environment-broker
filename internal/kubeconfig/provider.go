package kubeconfig

import (
	"context"
	"fmt"

	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	rbac "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const kcpNamespace = "kcp-system"

type SecretProvider struct {
	kcpK8sClient client.Client
}

func NewK8sClientFromSecretProvider(kcpK8sClient client.Client) *SecretProvider {
	return &SecretProvider{
		kcpK8sClient: kcpK8sClient,
	}
}

func (p *SecretProvider) KubeconfigForRuntimeID(runtimeId string) ([]byte, error) {
	kubeConfigSecret := &v1.Secret{}
	err := p.kcpK8sClient.Get(context.Background(), p.objectKey(runtimeId), kubeConfigSecret)
	if errors.IsNotFound(err) {
		return nil, NewNotFoundError(fmt.Sprintf("secret not found for runtime id %s", runtimeId))
	}
	if err != nil {
		return nil, fmt.Errorf("while getting secret from kcp for runtimeId=%s", runtimeId)
	}
	config, ok := kubeConfigSecret.Data["config"]
	if !ok {
		return nil, fmt.Errorf("while getting 'config' from secret from %s", p.objectKey(runtimeId))
	}
	if len(config) == 0 {
		return nil, fmt.Errorf("empty kubeconfig")
	}
	return config, nil
}

func (p *SecretProvider) objectKey(runtimeId string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: kcpNamespace,
		Name:      fmt.Sprintf("kubeconfig-%s", runtimeId),
	}
}

func (p *SecretProvider) K8sClientForRuntimeID(runtimeID string) (client.Client, error) {
	kubeconfig, err := p.KubeconfigForRuntimeID(runtimeID)
	if err != nil {
		return nil, err
	}
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("while creating rest config from kubeconfig")
	}

	k8sCli, err := client.New(restCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("while creating k8s client")
	}

	return k8sCli, nil
}

func (p *SecretProvider) K8sClientSetForRuntimeID(runtimeID string) (v12.CoreV1Interface, rbac.RbacV1Interface, error) {
	kubeconfig, err := p.KubeconfigForRuntimeID(runtimeID)
	if err != nil {
		return nil, nil, err
	}
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)

	if err != nil {
		return nil, nil, fmt.Errorf("while creating k8s client set - rest config from kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("while creating k8s client set")
	}

	return clientset.CoreV1(), clientset.RbacV1(), nil
}

type FakeProvider struct {
	c client.Client
}

func NewFakeK8sClientProvider(c client.Client) *FakeProvider {
	return &FakeProvider{c: c}
}

func (p *FakeProvider) K8sClientForRuntimeID(_ string) (client.Client, error) {
	if p.c == nil {
		return nil, fmt.Errorf("unable to get client")
	}
	return p.c, nil
}

func (p *FakeProvider) KubeconfigForRuntimeID(runtimeID string) ([]byte, error) {
	return []byte(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ca
    server: https://my.cluster
  name: <cluster-name>
contexts:
- context:
    cluster:  cname
    user:  cuser
  name:  cname
current-context:  cname
kind: Config
preferences: {}
users:
- name:  cuser
  user:
    token: some-token
`), nil
}

func (p *FakeProvider) K8sClientSetForRuntimeID(runtimeID string) (v12.CoreV1Interface, rbac.RbacV1Interface, error) {
	c := fake.NewSimpleClientset()
	_, err := c.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: machineryv1.ObjectMeta{Name: "kyma-system", Namespace: "kyma-system"},
	}, machineryv1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}
	coreClient := c.CoreV1()
	rbacClient := c.RbacV1()
	return coreClient, rbacClient, nil
}
