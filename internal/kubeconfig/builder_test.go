package kubeconfig

import (
	"fmt"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner/automock"

	"github.com/stretchr/testify/require"
)

const (
	globalAccountID = "d9d501c2-bdcb-49f2-8e86-1c4e05b90f5e"
	runtimeID       = "f7d634ae-4ce2-4916-be64-b6fb493155df"

	issuerURL = "https://example.com"
	clientID  = "c1id"
)

func TestBuilder_BuildFromProvisioner(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()

	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{
			RuntimeConfiguration: &schema.RuntimeConfig{
				Kubeconfig: skrKubeconfig(),
				ClusterConfig: &schema.GardenerConfig{
					OidcConfig: &schema.OIDCConfig{
						ClientID:       clientID,
						GroupsClaim:    "gclaim",
						IssuerURL:      issuerURL,
						SigningAlgs:    nil,
						UsernameClaim:  "uclaim",
						UsernamePrefix: "-",
					},
				},
			},
		}, nil)
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient, kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()))

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfig())
	})

	t.Run("provisioner client returned error", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{}, fmt.Errorf("cannot return kubeconfig"))
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient, kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()))
		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		_, err := builder.Build(instance)

		//then
		require.Error(t, err)
		require.Contains(t, err.Error(), "while fetching oidc data")
	})
}

func TestBuilder_BuildFromRuntimeResource(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()

	t.Run("new kubeconfig was built properly", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{
			RuntimeConfiguration: &schema.RuntimeConfig{
				Kubeconfig: skrKubeconfig(),
				ClusterConfig: &schema.GardenerConfig{
					OidcConfig: &schema.OIDCConfig{
						ClientID:       clientID,
						GroupsClaim:    "gclaim",
						IssuerURL:      issuerURL,
						SigningAlgs:    nil,
						UsernameClaim:  "uclaim",
						UsernamePrefix: "-",
					},
				},
			},
		}, nil)
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient, kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()))

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.Build(instance)

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newKubeconfig())
	})
}

func TestBuilder_BuildFromAdminKubeconfig(t *testing.T) {
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	t.Run("new kubeconfig was build properly", func(t *testing.T) {
		// given
		provisionerClient := &automock.Client{}
		provisionerClient.On("RuntimeStatus", globalAccountID, runtimeID).Return(schema.RuntimeStatus{
			RuntimeConfiguration: &schema.RuntimeConfig{
				Kubeconfig: skrKubeconfig(),
				ClusterConfig: &schema.GardenerConfig{
					OidcConfig: &schema.OIDCConfig{
						ClientID:       clientID,
						GroupsClaim:    "gclaim",
						IssuerURL:      issuerURL,
						SigningAlgs:    nil,
						UsernameClaim:  "uclaim",
						UsernamePrefix: "-",
					},
				},
			},
		}, nil)
		defer provisionerClient.AssertExpectations(t)

		builder := NewBuilder(provisionerClient, kcpClient, NewFakeKubeconfigProvider(skrKubeconfig()))

		instance := &internal.Instance{
			RuntimeID:       runtimeID,
			GlobalAccountID: globalAccountID,
		}

		// when
		kubeconfig, err := builder.BuildFromAdminKubeconfig(instance, adminKubeconfig())

		//then
		require.NoError(t, err)
		require.Equal(t, kubeconfig, newOwnClusterKubeconfig())
	})
}

func skrKubeconfig() *string {
	kc := `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9-token
users:
- name: shoot--kyma-dev--ac0d8d9-token
  user:
    token: DKPAe2Lt06a8dlUlE81kaWdSSDVSSf38x5PIj6cwQkqHMrw4UldsUr1guD6Thayw
`
	return &kc
}

func newKubeconfig() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--ac0d8d9
clusters:
- name: shoot--kyma-dev--ac0d8d9
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--ac0d8d9
  context:
    cluster: shoot--kyma-dev--ac0d8d9
    user: shoot--kyma-dev--ac0d8d9
users:
- name: shoot--kyma-dev--ac0d8d9
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`, issuerURL, clientID,
	)
}

func newOwnClusterKubeconfig() string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--admin
clusters:
- name: shoot--kyma-dev--admin
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--admin
  context:
    cluster: shoot--kyma-dev--admin
    user: shoot--kyma-dev--admin
users:
- name: shoot--kyma-dev--admin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url=%s"
      - "--oidc-client-id=%s"
      - "--oidc-extra-scope=email"
      - "--oidc-extra-scope=openid"
      command: kubectl-oidc_login
      installHint: |
        kubelogin plugin is required to proceed with authentication
        # Homebrew (macOS and Linux)
        brew install int128/kubelogin/kubelogin

        # Krew (macOS, Linux, Windows and ARM)
        kubectl krew install oidc-login

        # Chocolatey (Windows)
        choco install kubelogin
`, issuerURL, clientID,
	)
}

func adminKubeconfig() string {
	return `
---
apiVersion: v1
kind: Config
current-context: shoot--kyma-dev--admin
clusters:
- name: shoot--kyma-dev--admin
  cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURUSUZJQ0FURS0tLS0tCg==
    server: https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com
contexts:
- name: shoot--kyma-dev--admin
  context:
    cluster: shoot--kyma-dev--admin
    user: shoot--kyma-dev--admin-token
users:
- name: shoot--kyma-dev--admin-token
  user:
    token: DKPAe2Lt06a8dlUlE81kaWdSSDVSSf38x5PIj6cwQkqHMrw4UldsUr1guD6Thayw

`
}

func NewFakeKubeconfigProvider(content *string) *fakeKubeconfigProvider {
	return &fakeKubeconfigProvider{
		content: *content,
	}
}

type fakeKubeconfigProvider struct {
	content string
}

func (p *fakeKubeconfigProvider) KubeconfigForRuntimeID(_ string) ([]byte, error) {
	return []byte(p.content), nil
}
