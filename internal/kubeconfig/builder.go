package kubeconfig

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	AllowOrigins string
}

type Builder struct {
	kubeconfigProvider kubeconfigProvider
	kcpClient          client.Client
	useAdditionalOIDC  bool
	useMainOIDC        bool
}

type kubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeID string) ([]byte, error)
}

func NewBuilder(kcpClient client.Client, provider kubeconfigProvider, useAdditionalOIDC, useMainOIDC bool) *Builder {
	return &Builder{
		kcpClient:          kcpClient,
		kubeconfigProvider: provider,
		useAdditionalOIDC:  useAdditionalOIDC,
		useMainOIDC:        useMainOIDC,
	}
}

type kubeconfigData struct {
	ContextName   string
	CAData        string
	ServerURL     string
	OIDCIssuerURL string
	OIDCClientID  string
	Token         string
}

func (b *Builder) BuildFromAdminKubeconfigForBinding(runtimeID string, token string) (string, error) {
	adminKubeconfig, err := b.kubeconfigProvider.KubeconfigForRuntimeID(runtimeID)
	if err != nil {
		return "", err
	}

	kubeCfg, err := b.unmarshal(adminKubeconfig)
	if err != nil {
		return "", err
	}

	return b.parseTemplate(kubeconfigData{
		ContextName: kubeCfg.CurrentContext,
		CAData:      kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:   kubeCfg.Clusters[0].Cluster.Server,
		Token:       token,
	}, kubeconfigTemplateForKymaBindings)
}

func (b *Builder) BuildFromAdminKubeconfig(instance *internal.Instance, adminKubeconfig string) (string, error) {
	if instance.RuntimeID == "" {
		return "", fmt.Errorf("RuntimeID must not be empty")
	}
	issuerURL, clientID, err := b.getOidcDataFromRuntimeResource(instance.RuntimeID)
	if err != nil {
		return "", fmt.Errorf("while fetching oidc data: %w", err)
	}

	var kubeconfigContent []byte
	if adminKubeconfig == "" {
		kubeconfigContent, err = b.kubeconfigProvider.KubeconfigForRuntimeID(instance.RuntimeID)
		if err != nil {
			return "", err
		}
	} else {
		kubeconfigContent = []byte(adminKubeconfig)
	}

	kubeCfg, err := b.unmarshal(kubeconfigContent)
	if err != nil {
		return "", fmt.Errorf("during unmarshal invocation: %w", err)
	}

	return b.parseTemplate(kubeconfigData{
		ContextName:   kubeCfg.CurrentContext,
		CAData:        kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:     kubeCfg.Clusters[0].Cluster.Server,
		OIDCIssuerURL: issuerURL,
		OIDCClientID:  clientID,
	}, kubeconfigTemplate)
}

func (b *Builder) unmarshal(kubeconfigContent []byte) (*kubeconfig, error) {
	var kubeCfg kubeconfig

	err := yaml.Unmarshal(kubeconfigContent, &kubeCfg)
	if err != nil {
		return nil, fmt.Errorf("while unmarshaling kubeconfig: %w", err)
	}
	if err := b.validKubeconfig(kubeCfg); err != nil {
		return nil, fmt.Errorf("while validation kubeconfig fetched by provisioner: %w", err)
	}
	return &kubeCfg, nil
}

func (b *Builder) Build(instance *internal.Instance) (string, error) {
	return b.BuildFromAdminKubeconfig(instance, "")
}

func (b *Builder) GetServerURL(runtimeID string) (string, error) {
	if runtimeID == "" {
		return "", fmt.Errorf("runtimeID must not be empty")
	}
	var kubeCfg kubeconfig
	kubeconfigContent, err := b.kubeconfigProvider.KubeconfigForRuntimeID(runtimeID)
	if err != nil {
		return "", err
	}
	err = yaml.Unmarshal(kubeconfigContent, &kubeCfg)
	if err != nil {
		return "", fmt.Errorf("while unmarshaling kubeconfig: %w", err)
	}
	if err := b.validKubeconfig(kubeCfg); err != nil {
		return "", fmt.Errorf("while validation kubeconfig fetched by provisioner: %w", err)
	}
	return kubeCfg.Clusters[0].Cluster.Server, nil
}

func (b *Builder) parseTemplate(payload kubeconfigData, templateName string) (string, error) {
	var result bytes.Buffer
	t := template.New("kubeconfigParser")
	t, err := t.Parse(templateName)
	if err != nil {
		return "", fmt.Errorf("while parsing kubeconfig template: %w", err)
	}

	err = t.Execute(&result, payload)
	if err != nil {
		return "", fmt.Errorf("while executing kubeconfig template: %w", err)
	}
	return result.String(), nil
}

func (b *Builder) validKubeconfig(kc kubeconfig) error {
	if kc.CurrentContext == "" {
		return fmt.Errorf("current context is empty")
	}
	if len(kc.Clusters) == 0 {
		return fmt.Errorf("there are no defined clusters")
	}
	if kc.Clusters[0].Cluster.CertificateAuthorityData == "" || kc.Clusters[0].Cluster.Server == "" {
		return fmt.Errorf("there are no cluster certificate or server info")
	}

	return nil
}

func (b *Builder) getOidcDataFromRuntimeResource(id string) (string, string, error) {
	var runtime imv1.Runtime
	var issuerURL, clientID string
	err := b.kcpClient.Get(context.Background(), client.ObjectKey{Name: id, Namespace: kcpNamespace}, &runtime)
	if err != nil {
		return "", "", err
	}
	if !b.useMainOIDC && b.useAdditionalOIDC {
		if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig == nil {
			return "", "", fmt.Errorf("Runtime Resource contains no OIDC config")
		}
		if (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].IssuerURL == nil {
			return "", "", fmt.Errorf("Runtime Resource contains an empty OIDC issuer URL")
		}
		if (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID == nil {
			return "", "", fmt.Errorf("Runtime Resource contains an empty OIDC client ID")
		}
		issuerURL = *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].IssuerURL
		clientID = *(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID
	} else {
		if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL == nil {
			return "", "", fmt.Errorf("Runtime Resource contains an empty OIDC issuer URL")
		}
		if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID == nil {
			return "", "", fmt.Errorf("Runtime Resource contains an empty OIDC client ID")
		}
		issuerURL = *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL
		clientID = *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID
	}
	return issuerURL, clientID, nil
}
