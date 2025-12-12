package kubeconfig

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	AllowOrigins string
}

type Builder struct {
	kubeconfigProvider kubeconfigProvider
	kcpClient          client.Client
}

type kubeconfigProvider interface {
	KubeconfigForRuntimeID(runtimeID string) ([]byte, error)
}

func NewBuilder(kcpClient client.Client, provider kubeconfigProvider) *Builder {
	return &Builder{
		kcpClient:          kcpClient,
		kubeconfigProvider: provider,
	}
}

type kubeconfigData struct {
	ContextName string
	CAData      string
	ServerURL   string
	OIDCConfigs []OIDCConfig
	Token       string
}

type OIDCConfig struct {
	Name      string
	IssuerURL string
	ClientID  string
}

func (b *Builder) BuildFromAdminKubeconfigForBinding(runtimeID string, token string, clusterName string) (string, error) {
	adminKubeconfig, err := b.kubeconfigProvider.KubeconfigForRuntimeID(runtimeID)
	if err != nil {
		return "", err
	}

	kubeCfg, err := b.unmarshal(adminKubeconfig)
	if err != nil {
		return "", err
	}

	return b.parseTemplate(kubeconfigData{
		ContextName: clusterName,
		CAData:      kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:   kubeCfg.Clusters[0].Cluster.Server,
		Token:       token,
	}, kubeconfigTemplateForKymaBindings)
}

func (b *Builder) BuildFromAdminKubeconfig(instance *internal.Instance, adminKubeconfig string) (string, error) {
	if instance.RuntimeID == "" {
		return "", fmt.Errorf("RuntimeID must not be empty")
	}

	var kubeconfigContent []byte
	var err error
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

	OIDCConfigs, err := b.getOidcDataFromRuntimeResource(instance.RuntimeID, instance.Parameters.Parameters.Name)
	if err != nil {
		return "", fmt.Errorf("while fetching oidc data: %w", err)
	}

	return b.parseTemplate(kubeconfigData{
		ContextName: instance.Parameters.Parameters.Name,
		CAData:      kubeCfg.Clusters[0].Cluster.CertificateAuthorityData,
		ServerURL:   kubeCfg.Clusters[0].Cluster.Server,
		OIDCConfigs: OIDCConfigs,
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

func (b *Builder) getOidcDataFromRuntimeResource(id string, currentContext string) ([]OIDCConfig, error) {
	var runtime imv1.Runtime
	var oidcConfigs []OIDCConfig
	err := b.kcpClient.Get(context.Background(), client.ObjectKey{Name: id, Namespace: kcpNamespace}, &runtime)
	if err != nil {
		return nil, err
	}
	additionalConfigs := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	if additionalConfigs == nil {
		return nil, fmt.Errorf("Runtime Resource contains no additional OIDC config")
	}
	for i, config := range *additionalConfigs {
		if config.IssuerURL == nil {
			return nil, fmt.Errorf("Runtime Resource contains an empty OIDC issuer URL")
		}
		if config.ClientID == nil {
			return nil, fmt.Errorf("Runtime Resource contains an empty OIDC client ID")
		}
		name := currentContext
		if i > 0 {
			name = fmt.Sprintf("%s-%d", currentContext, i+1)
		}
		oidcConfigs = append(oidcConfigs, OIDCConfig{
			Name:      name,
			IssuerURL: *config.IssuerURL,
			ClientID:  *config.ClientID,
		})
	}
	return oidcConfigs, nil
}
