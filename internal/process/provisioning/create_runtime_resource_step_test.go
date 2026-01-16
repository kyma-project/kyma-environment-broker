package provisioning

import (
	"context"
	"encoding/base64"
	"os"
	"reflect"
	"strings"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/customresources"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	SecretBindingName = "gardener-secret"
	OperationID       = "operation-01"
)

var runtimeAdministrators = []string{"admin1@test.com", "admin2@test.com"}

var defaultNetworking = imv1.Networking{
	Nodes:    networking.DefaultNodesCIDR,
	Pods:     networking.DefaultPodsCIDR,
	Services: networking.DefaultServicesCIDR,
	// TODO: remove after KIM is handling this properly
	Type: ptr.String("calico"),
}

var defaultOIDSConfig = pkg.OIDCConfigDTO{
	ClientID:       "client-id-default",
	GroupsClaim:    "gc-default",
	GroupsPrefix:   "-",
	IssuerURL:      "issuer-url-default",
	SigningAlgs:    []string{"sa-default"},
	UsernameClaim:  "uc-default",
	UsernamePrefix: "up-default",
}

func TestCreateRuntimeResourceStep_AllCustom(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		OIDCConfigDTO: &pkg.OIDCConfigDTO{
			ClientID:       "client-id-custom",
			GroupsClaim:    "gc-custom",
			IssuerURL:      "issuer-url-custom",
			SigningAlgs:    []string{"sa-custom"},
			UsernameClaim:  "uc-custom",
			UsernamePrefix: "up-custom",
			RequiredClaims: []string{"claim=value", "claim2=value2=value2", "claim3==value3", "claim4=value4=", "claim5=,value5", "claim6=="},
			GroupsPrefix:   "SAPSE:",
		},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	expectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("client-id-custom"),
			GroupsClaim:    ptr.String("gc-custom"),
			IssuerURL:      ptr.String("issuer-url-custom"),
			SigningAlgs:    []string{"sa-custom"},
			UsernameClaim:  ptr.String("uc-custom"),
			UsernamePrefix: ptr.String("up-custom"),
			RequiredClaims: map[string]string{
				"claim":  "value",
				"claim2": "value2=value2",
				"claim3": "=value3",
				"claim4": "value4=",
				"claim5": ",value5",
				"claim6": "=",
			},
			GroupsPrefix: ptr.String("SAPSE:"),
		},
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.RequiredClaims)
	assert.Equal(t, expectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func TestCreateRuntimeResourceStep_AllCustomWithOIDCList(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		List: []pkg.OIDCConfigDTO{
			{
				ClientID:       "client-id-custom",
				GroupsClaim:    "gc-custom",
				IssuerURL:      "issuer-url-custom",
				SigningAlgs:    []string{"sa-custom"},
				UsernameClaim:  "uc-custom",
				UsernamePrefix: "up-custom",
				RequiredClaims: []string{"claim=value"},
				GroupsPrefix:   "SAPSE:",
			},
		},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	expectedAdditionalOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("client-id-custom"),
			GroupsClaim:    ptr.String("gc-custom"),
			IssuerURL:      ptr.String("issuer-url-custom"),
			SigningAlgs:    []string{"sa-custom"},
			UsernameClaim:  ptr.String("uc-custom"),
			UsernamePrefix: ptr.String("up-custom"),
			RequiredClaims: map[string]string{
				"claim": "value",
			},
			GroupsPrefix: ptr.String("SAPSE:"),
		},
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.RequiredClaims)
	assert.Equal(t, expectedAdditionalOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func TestCreateRuntimeResourceStep_HandleMultipleAdditionalOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		List: []pkg.OIDCConfigDTO{
			{
				ClientID:       "first-client-id-custom",
				GroupsClaim:    "first-gc-custom",
				IssuerURL:      "first-issuer-url-custom",
				SigningAlgs:    []string{"first-sa-custom"},
				UsernameClaim:  "first-uc-custom",
				UsernamePrefix: "first-up-custom",
			},
			{
				ClientID:       "second-client-id-custom",
				GroupsClaim:    "second-gc-custom",
				IssuerURL:      "second-issuer-url-custom",
				SigningAlgs:    []string{"second-sa-custom"},
				UsernameClaim:  "second-uc-custom",
				UsernamePrefix: "second-up-custom",
			},
		},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	firstExpectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("first-client-id-custom"),
			GroupsClaim:    ptr.String("first-gc-custom"),
			IssuerURL:      ptr.String("first-issuer-url-custom"),
			SigningAlgs:    []string{"first-sa-custom"},
			UsernameClaim:  ptr.String("first-uc-custom"),
			UsernamePrefix: ptr.String("first-up-custom"),
			GroupsPrefix:   ptr.String("-"),
		},
	}
	secondExpectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("second-client-id-custom"),
			GroupsClaim:    ptr.String("second-gc-custom"),
			IssuerURL:      ptr.String("second-issuer-url-custom"),
			SigningAlgs:    []string{"second-sa-custom"},
			UsernameClaim:  ptr.String("second-uc-custom"),
			UsernamePrefix: ptr.String("second-up-custom"),
			GroupsPrefix:   ptr.String("-"),
		},
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))
	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.Equal(t, firstExpectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
	assert.Equal(t, secondExpectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[1])
}

func TestCreateRuntimeResourceStep_OIDC_MixedCustom(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		OIDCConfigDTO: &pkg.OIDCConfigDTO{
			ClientID:      "client-id-custom",
			GroupsClaim:   "gc-custom",
			IssuerURL:     "issuer-url-custom",
			UsernameClaim: "uc-custom",
		},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	expectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("client-id-custom"),
			GroupsClaim:    ptr.String("gc-custom"),
			IssuerURL:      ptr.String("issuer-url-custom"),
			SigningAlgs:    []string{"sa-default"},
			UsernameClaim:  ptr.String("uc-custom"),
			UsernamePrefix: ptr.String("up-default"),
			GroupsPrefix:   ptr.String("-"),
		},
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, expectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func TestCreateRuntimeResourceStep_HandleEmptyOIDCList(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		List: []pkg.OIDCConfigDTO{},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.NotNil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
	assert.Len(t, *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 0)
}

func TestCreateRuntimeResourceStep_HandleNotNilOIDCWithoutListOrObject(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{}
	assertInsertions(t, memoryStorage, instance, operation)
	expectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("client-id-default"),
			GroupsClaim:    ptr.String("gc-default"),
			IssuerURL:      ptr.String("issuer-url-default"),
			SigningAlgs:    []string{"sa-default"},
			UsernameClaim:  ptr.String("uc-default"),
			UsernamePrefix: ptr.String("up-default"),
			GroupsPrefix:   ptr.String("-"),
		},
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.NotNil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
	assert.Equal(t, expectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func TestCreateRuntimeResourceStep_HandleOIDCWithJwks(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		OIDCConfigDTO: &pkg.OIDCConfigDTO{
			ClientID:         "client-id-custom",
			GroupsClaim:      "gc-custom",
			IssuerURL:        "issuer-url-custom",
			SigningAlgs:      []string{"sa-custom"},
			UsernameClaim:    "uc-custom",
			UsernamePrefix:   "up-custom",
			EncodedJwksArray: "andrcy10b2tlbi1kZWZhdWx0",
		},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	expectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("client-id-custom"),
			GroupsClaim:    ptr.String("gc-custom"),
			IssuerURL:      ptr.String("issuer-url-custom"),
			SigningAlgs:    []string{"sa-custom"},
			UsernameClaim:  ptr.String("uc-custom"),
			UsernamePrefix: ptr.String("up-custom"),
			GroupsPrefix:   ptr.String("-"),
		},
		JWKS: func() []byte {
			b, _ := base64.StdEncoding.DecodeString("andrcy10b2tlbi1kZWZhdWx0")
			return b
		}(),
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.NotNil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
	assert.Equal(t, expectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func TestCreateRuntimeResourceStep_HandleAdditionalOIDCWithJWKS(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		List: []pkg.OIDCConfigDTO{
			{
				ClientID:         "first-client-id-custom",
				GroupsClaim:      "first-gc-custom",
				IssuerURL:        "first-issuer-url-custom",
				SigningAlgs:      []string{"first-sa-custom"},
				UsernameClaim:    "first-uc-custom",
				UsernamePrefix:   "first-up-custom",
				EncodedJwksArray: "andrcy10b2tlbi1kZWZhdWx0",
			},
			{
				ClientID:         "second-client-id-custom",
				GroupsClaim:      "second-gc-custom",
				IssuerURL:        "second-issuer-url-custom",
				SigningAlgs:      []string{"second-sa-custom"},
				UsernameClaim:    "second-uc-custom",
				UsernamePrefix:   "second-up-custom",
				EncodedJwksArray: "",
			},
			{
				ClientID:         "third-client-id-custom",
				GroupsClaim:      "third-gc-custom",
				IssuerURL:        "third-issuer-url-custom",
				SigningAlgs:      []string{"third-sa-custom"},
				UsernameClaim:    "third-uc-custom",
				UsernamePrefix:   "third-up-custom",
				EncodedJwksArray: "-",
			},
		},
	}
	assertInsertions(t, memoryStorage, instance, operation)
	firstExpectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("first-client-id-custom"),
			GroupsClaim:    ptr.String("first-gc-custom"),
			IssuerURL:      ptr.String("first-issuer-url-custom"),
			SigningAlgs:    []string{"first-sa-custom"},
			UsernameClaim:  ptr.String("first-uc-custom"),
			UsernamePrefix: ptr.String("first-up-custom"),
			GroupsPrefix:   ptr.String("-"),
		},
		JWKS: func() []byte {
			b, _ := base64.StdEncoding.DecodeString("andrcy10b2tlbi1kZWZhdWx0")
			return b
		}(),
	}
	secondExpectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("second-client-id-custom"),
			GroupsClaim:    ptr.String("second-gc-custom"),
			IssuerURL:      ptr.String("second-issuer-url-custom"),
			SigningAlgs:    []string{"second-sa-custom"},
			UsernameClaim:  ptr.String("second-uc-custom"),
			UsernamePrefix: ptr.String("second-up-custom"),
			GroupsPrefix:   ptr.String("-"),
		},
	}
	thirdExpectedOIDCConfig := imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       ptr.String("third-client-id-custom"),
			GroupsClaim:    ptr.String("third-gc-custom"),
			IssuerURL:      ptr.String("third-issuer-url-custom"),
			SigningAlgs:    []string{"third-sa-custom"},
			UsernameClaim:  ptr.String("third-uc-custom"),
			UsernamePrefix: ptr.String("third-up-custom"),
			GroupsPrefix:   ptr.String("-"),
		},
	}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))
	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.Equal(t, firstExpectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
	assert.Equal(t, secondExpectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[1])
	assert.Equal(t, thirdExpectedOIDCConfig, (*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[2])
}

func TestCreateRuntimeResourceStep_FailureToleranceForTrial(t *testing.T) {
	// given
	assert.NoError(t, imv1.AddToScheme(scheme.Scheme))
	memoryStorage := storage.NewMemoryStorage()

	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = "zone"
	inputConfig.DefaultTrialProvider = "AWS"

	instance, operation := fixInstanceAndOperation(broker.TrialPlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, _, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.ControlPlane)
}

func TestCreateRuntimeResourceStep_FailureToleranceForCommercial(t *testing.T) {
	// given
	assert.NoError(t, imv1.AddToScheme(scheme.Scheme))
	memoryStorage := storage.NewMemoryStorage()

	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = "zone"

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, _, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))
}

func TestCreateRuntimeResourceStep_FailureToleranceForCommercialWithNoConfig(t *testing.T) {
	// given
	assert.NoError(t, imv1.AddToScheme(scheme.Scheme))
	memoryStorage := storage.NewMemoryStorage()

	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = ""

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, _, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.ControlPlane)
}

func TestCreateRuntimeResourceStep_FailureToleranceForCommercialWithConfiguredNode(t *testing.T) {
	// given
	assert.NoError(t, imv1.AddToScheme(scheme.Scheme))
	memoryStorage := storage.NewMemoryStorage()

	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = "node"

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "westeurope", "platform-region", inputConfig, pkg.AWS)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, _, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, "node", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))
}

// Actual creation tests

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_EnforceSeed(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.Parameters.ColocateControlPlane = ptr.Bool(true)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityEgressEnabled(t, runtime)

	assert.True(t, *runtime.Spec.Shoot.EnforceSeedLocation)
	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assert.Equal(t, SecretBindingName, runtime.Spec.Shoot.SecretBindingName)
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
	assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))
	assertDefaultNetworking(t, runtime.Spec.Shoot.Networking)

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_DisableEnterpriseFilter(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.ErsContext.LicenseType = ptr.String("PARTNER")
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)

	assertSecurityEgressDisabled(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assert.Equal(t, SecretBindingName, runtime.Spec.Shoot.SecretBindingName)
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
	assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))
	assertDefaultNetworking(t, runtime.Spec.Shoot.Networking)

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_NetworkFilter(t *testing.T) {
	// given
	for _, testCase := range []struct {
		name                      string
		planID                    string
		cloudProvider             pkg.CloudProvider
		licenseType               string
		ingressFilteringParameter *bool

		expectedEgressResult  bool
		expectedIngressResult bool
	}{
		{"External - not supported plan - turn on", broker.SapConvergedCloudPlanID, pkg.SapConvergedCloud, "CUSTOMER", ptr.Bool(true), false, false},
		{"Internal - not supported plan - turn on", broker.SapConvergedCloudPlanID, pkg.SapConvergedCloud, "NON-CUSTOMER", ptr.Bool(true), true, false},
		{"External - not supported plan", broker.SapConvergedCloudPlanID, pkg.SapConvergedCloud, "CUSTOMER", nil, false, false},
		{"External - AWS", broker.AWSPlanID, pkg.AWS, "CUSTOMER", nil, false, false},
		{"Internal - AWS - no parameter", broker.AWSPlanID, pkg.AWS, "NON-CUSTOMER", nil, true, false},
		{"Internal - AWS - turn on", broker.AWSPlanID, pkg.AWS, "NON-CUSTOMER", ptr.Bool(true), true, true},
		{"Internal - Azure - turn on", broker.AzurePlanID, pkg.Azure, "NON-CUSTOMER", ptr.Bool(true), true, false},
		{"Internal - AWS - turn off", broker.AWSPlanID, pkg.AWS, "NON-CUSTOMER", ptr.Bool(false), true, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// when
			memoryStorage := storage.NewMemoryStorage()
			err := imv1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)

			cli := getClientForTests(t)

			infrastructureManagerConfig := broker.InfrastructureManager{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone",
				DefaultGardenerShootPurpose: provider.PurposeProduction, IngressFilteringPlans: []string{"aws"}}
			instance, operation := fixInstanceAndOperation(testCase.planID, "hyperscaler-region", "platform-region", infrastructureManagerConfig, testCase.cloudProvider)
			assertInsertions(t, memoryStorage, instance, operation)

			operation.ProvisioningParameters.ErsContext.LicenseType = ptr.String(testCase.licenseType)
			operation.ProvisioningParameters.Parameters.IngressFiltering = testCase.ingressFilteringParameter
			operation.CloudProvider = string(testCase.cloudProvider)
			step := NewCreateRuntimeResourceStep(memoryStorage, cli, infrastructureManagerConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))
			_, repeat, err := step.Run(operation, fixLogger())

			// then
			assert.NoError(t, err)
			assert.Zero(t, repeat)

			runtime := imv1.Runtime{}
			err = cli.Get(context.Background(), client.ObjectKey{
				Namespace: "kyma-system",
				Name:      operation.RuntimeID,
			}, &runtime)
			assert.NoError(t, err)
			assert.Equal(t, runtime.Name, operation.RuntimeID)
			assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

			assertLabelsKIMDriven(t, operation, runtime)

			assertSecurityWithNetworkFilter(t, runtime, testCase.expectedEgressResult, testCase.expectedIngressResult)

			assertDefaultNetworking(t, runtime.Spec.Shoot.Networking)

			_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
			assert.NoError(t, err)

		})
	}
}

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_DefaultAdmin(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.Parameters.RuntimeAdministrators = nil
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityWithDefaultAdministrator(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assert.Equal(t, SecretBindingName, runtime.Spec.Shoot.SecretBindingName)
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
	assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))
	assertDefaultNetworking(t, runtime.Spec.Shoot.Networking)

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZoneWithNetworking(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.Parameters.Networking = &pkg.NetworkingDTO{
		NodesCidr:    "192.168.48.0/20",
		PodsCidr:     ptr.String("10.104.0.0/24"),
		ServicesCidr: ptr.String("10.105.0.0/24"),
	}

	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityEgressEnabled(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkersWithVolume(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 3, 0, 3, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"}, "80Gi", "gp3")
	assertNetworking(t, imv1.Networking{
		Nodes:    "192.168.48.0/20",
		Pods:     "10.104.0.0/24",
		Services: "10.105.0.0/24",
		// TODO remove after KIM is handling this properly
		Type: ptr.String("calico"),
	}, runtime.Spec.Shoot.Networking)

	assert.Equal(t, "any-string", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZone(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityEgressEnabled(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 3, 0, 3, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
	assert.Equal(t, "any-string", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_DualStack(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	dualStackEnabled := true
	operation.ProvisioningParameters.Parameters.Networking = &pkg.NetworkingDTO{
		NodesCidr:    "192.168.48.0/20",
		PodsCidr:     ptr.String("10.104.0.0/24"),
		ServicesCidr: ptr.String("10.105.0.0/24"),
		DualStack:    &dualStackEnabled,
	}

	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, newTestProviderSpecWithDualStack())

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.NotNil(t, runtime.Spec.Shoot.Networking.DualStack)
	assert.True(t, *runtime.Spec.Shoot.Networking.DualStack)

	expectedNetworking := imv1.Networking{
		Nodes:     "192.168.48.0/20",
		Pods:      "10.104.0.0/24",
		Services:  "10.105.0.0/24",
		DualStack: &dualStackEnabled,
		Type:      ptr.String("calico"),
	}
	assertNetworking(t, expectedNetworking, runtime.Spec.Shoot.Networking)
}

func TestCreateRuntimeResourceStep_DualStackDisabled(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	dualStackDisabled := false
	operation.ProvisioningParameters.Parameters.Networking = &pkg.NetworkingDTO{
		NodesCidr: "192.168.48.0/20",
		DualStack: &dualStackDisabled,
	}

	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, newTestProviderSpecWithDualStack())

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)

	assert.NotNil(t, runtime.Spec.Shoot.Networking.DualStack)
	assert.False(t, *runtime.Spec.Shoot.Networking.DualStack)
}

func TestCreateRuntimeResourceStep_DualStackNotSet(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.Parameters.Networking = &pkg.NetworkingDTO{
		NodesCidr: "192.168.48.0/20",
	}

	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, newTestProviderSpecWithDualStack())

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Networking.DualStack)
}

func TestCreateRuntimeResourceStep_DualStackIgnoredForUnsupportedPlan(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region", inputConfig, pkg.Azure)
	operation.ProvisioningParameters.Parameters.Networking = &pkg.NetworkingDTO{
		NodesCidr:    "192.168.48.0/20",
		PodsCidr:     ptr.String("10.104.0.0/24"),
		ServicesCidr: ptr.String("10.105.0.0/24"),
		DualStack:    ptr.Bool(true),
	}

	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, newTestProviderSpecWithDualStack())

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Nil(t, runtime.Spec.Shoot.Networking.DualStack)

	expectedNetworking := imv1.Networking{
		Nodes:     "192.168.48.0/20",
		Pods:      "10.104.0.0/24",
		Services:  "10.105.0.0/24",
		DualStack: nil,
		Type:      ptr.String("calico"),
	}
	assertNetworking(t, expectedNetworking, runtime.Spec.Shoot.Networking)
}

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: false, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "zone"}

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, operation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityEgressEnabled(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_WithRetry(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: false, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "zone"}

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, operation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityEgressEnabled(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	// then retry
	_, repeat, err = step.Run(operation, fixLogger())
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, operation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabelsKIMDriven(t, operation, runtime)
	assertSecurityEgressEnabled(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_SapConvergedCloud(t *testing.T) {

	for _, testCase := range []struct {
		name                string
		gotProvider         pkg.CloudProvider
		expectedZonesCount  int
		expectedProvider    string
		expectedMachineType string
		expectedRegion      string
		possibleZones       []string
	}{
		{"Single zone", pkg.SapConvergedCloud, 1, "openstack", "g_c2_m8", "eu-de-1", []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
		{"Multi zone", pkg.SapConvergedCloud, 3, "openstack", "g_c2_m8", "eu-de-1", []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			memoryStorage := storage.NewMemoryStorage()
			err := imv1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)
			inputConfig := broker.InfrastructureManager{MultiZoneCluster: testCase.expectedZonesCount > 1, ControlPlaneFailureTolerance: "zone"}
			instance, operation := fixInstanceAndOperation(broker.SapConvergedCloudPlanID, "", "platform-region", inputConfig, testCase.gotProvider)
			assertInsertions(t, memoryStorage, instance, operation)

			cli := getClientForTests(t)
			step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))
			// when
			gotOperation, repeat, err := step.Run(operation, fixLogger())

			// then
			assert.NoError(t, err)
			assert.Zero(t, repeat)
			assert.Equal(t, domain.InProgress, gotOperation.State)

			runtime := imv1.Runtime{}
			err = cli.Get(context.Background(), client.ObjectKey{
				Namespace: "kyma-system",
				Name:      operation.RuntimeID,
			}, &runtime)
			assert.NoError(t, err)
			assert.Equal(t, operation.RuntimeID, runtime.Name)
			assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])
			assert.Equal(t, testCase.expectedProvider, runtime.Spec.Shoot.Provider.Type)
			assert.Nil(t, runtime.Spec.Shoot.Provider.Workers[0].Volume)
			assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, testCase.expectedMachineType, 20, 3, testCase.expectedZonesCount, 0, testCase.expectedZonesCount, testCase.possibleZones)

			assert.Equal(t, "zone", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))

		})
	}
}

func TestCreateRuntimeResourceStep_Defaults_Freemium(t *testing.T) {

	for _, testCase := range []struct {
		name                string
		gotProvider         pkg.CloudProvider
		expectedProvider    string
		expectedMachineType string
		expectedRegion      string
		possibleZones       []string
	}{
		/**
		  zone provider is mocked, always returns: a, b, c
		*/
		{"azure", pkg.Azure, "azure", "Standard_D4s_v5", "westeurope", []string{"a", "b", "c"}},
		{"aws", pkg.AWS, "aws", "m5.xlarge", "westeurope", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			memoryStorage := storage.NewMemoryStorage()
			err := imv1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)
			inputConfig := broker.InfrastructureManager{MultiZoneCluster: true}
			instance, operation := fixInstanceAndOperation(broker.FreemiumPlanID, "", "platform-region", inputConfig, testCase.gotProvider)
			assertInsertions(t, memoryStorage, instance, operation)

			cli := getClientForTests(t)
			step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

			// when
			gotOperation, repeat, err := step.Run(operation, fixLogger())

			// then
			assert.NoError(t, err)
			assert.Zero(t, repeat)
			assert.Equal(t, domain.InProgress, gotOperation.State)

			runtime := imv1.Runtime{}
			err = cli.Get(context.Background(), client.ObjectKey{
				Namespace: "kyma-system",
				Name:      operation.RuntimeID,
			}, &runtime)
			assert.NoError(t, err)
			assert.Equal(t, operation.RuntimeID, runtime.Name)
			assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])
			assert.Equal(t, testCase.expectedProvider, runtime.Spec.Shoot.Provider.Type)
			assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, testCase.expectedMachineType, 1, 1, 1, 0, 1, testCase.possibleZones)

			assert.Nil(t, runtime.Spec.Shoot.ControlPlane)
		})
	}
}

func TestCreateRuntimeResourceStep_AWS_ZonesDiscovery(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	machineType := "m6i.large"
	operation.ProvisioningParameters.Parameters.MachineType = &machineType
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "worker-1",
			MachineType:   "m6i.large",
			HAZones:       true,
			AutoScalerMin: 3,
			AutoScalerMax: 20,
		},
		{
			Name:          "worker-2",
			MachineType:   "m5.large",
			HAZones:       false,
			AutoScalerMin: 1,
			AutoScalerMax: 1,
		},
	}
	operation.DiscoveredZones = map[string][]string{
		"m6i.large": {"zone-d", "zone-e", "zone-f", "zone-h"},
		"m5.large":  {"zone-i", "zone-j", "zone-k", "zone-l"},
	}
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, workers.NewProvider(broker.InfrastructureManager{}, fixture.NewProviderSpecWithZonesDiscovery(t, true)), fixture.NewProviderSpecWithZonesDiscovery(t, true))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	require.NoError(t, err)

	require.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 3)
	assert.Subset(t, []string{"zone-d", "zone-e", "zone-f", "zone-h"}, runtime.Spec.Shoot.Provider.Workers[0].Zones)

	require.NotNil(t, runtime.Spec.Shoot.Provider.AdditionalWorkers)
	assert.Len(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers, 2)

	assert.Len(t, (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[0].Zones, 3)
	assert.Subset(t, []string{"zone-d", "zone-e", "zone-f", "zone-h"}, (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[0].Zones)

	assert.Len(t, (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[1].Zones, 1)
	assert.Subset(t, []string{"zone-i", "zone-j", "zone-k", "zone-l"}, (*runtime.Spec.Shoot.Provider.AdditionalWorkers)[1].Zones)
}

func TestCreateRuntimeResourceStep_Free_ZonesDiscovery(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)
	inputConfig := broker.InfrastructureManager{MultiZoneCluster: true, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	instance, operation := fixInstanceAndOperation(broker.FreemiumPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.DiscoveredZones = map[string][]string{
		"m5.xlarge": {"zone-i", "zone-j", "zone-k", "zone-l"},
	}
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, workers.NewProvider(broker.InfrastructureManager{}, fixture.NewProviderSpecWithZonesDiscovery(t, true)), fixture.NewProviderSpecWithZonesDiscovery(t, true))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	require.NoError(t, err)

	require.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 1)
	assert.Subset(t, []string{"zone-i", "zone-j", "zone-k", "zone-l"}, runtime.Spec.Shoot.Provider.Workers[0].Zones)
}

func TestCreateRuntimeResourceStep_AdditionalWorkersNilHandling(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = nil
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)

	assert.NotNil(t, runtime.Spec.Shoot.Provider.AdditionalWorkers)
	assert.Empty(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers)
}

func TestCreateRuntimeResourceStep_AdditionalWorkersEmptyHandling(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	inputConfig := broker.InfrastructureManager{
		MultiZoneCluster: true,
	}
	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region", inputConfig, pkg.AWS)
	operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{}
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage, cli, inputConfig, defaultOIDSConfig, &workers.Provider{}, fixture.NewProviderSpecWithZonesDiscovery(t, false))

	// when
	_, repeat, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      operation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, runtime.Name, operation.RuntimeID)

	assert.NotNil(t, runtime.Spec.Shoot.Provider.AdditionalWorkers)
	assert.Empty(t, *runtime.Spec.Shoot.Provider.AdditionalWorkers)
}

// testing auxiliary functions

func Test_Defaults(t *testing.T) {
	nilToDefaultString := DefaultIfParamNotSet("default value", nil)
	nonDefaultString := DefaultIfParamNotSet("default value", ptr.String("initial value"))

	nilToDefaultInt := DefaultIfParamNotSet(42, nil)
	nonDefaultInt := DefaultIfParamNotSet(42, ptr.Integer(7))

	assert.Equal(t, "initial value", nonDefaultString)
	assert.Equal(t, "default value", nilToDefaultString)
	assert.Equal(t, 42, nilToDefaultInt)
	assert.Equal(t, 7, nonDefaultInt)
}

func Test_IsIngressFilteringEnabled(t *testing.T) {

	for _, testCase := range []struct {
		name            string
		externalAccount bool
		planID          string

		expectedResult bool
	}{
		// given
		{"SapConvergedCloud", false, broker.SapConvergedCloudPlanID, false},
		{"internal GCP", false, broker.GCPPlanID, true},
		{"internal Azure", false, broker.AzurePlanID, false},
		{"external GCP", true, broker.GCPPlanID, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			imConfig := broker.InfrastructureManager{IngressFilteringPlans: []string{"aws", "gcp"}}
			// when
			result := steps.IsIngressFilteringEnabled(testCase.planID, imConfig, testCase.externalAccount)
			// then
			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}

func Test_IsExternalCustomer(t *testing.T) {

	for _, testCase := range []struct {
		name       string
		ersContext internal.ERSContext

		expectedResult bool
	}{
		// given
		{"nil license", internal.ERSContext{LicenseType: nil}, false},
		{"CUSTOMER license", internal.ERSContext{LicenseType: ptr.String("CUSTOMER")}, true},
		{"TRIAL license", internal.ERSContext{LicenseType: ptr.String("TRIAL")}, true},
		{"PARTNER license", internal.ERSContext{LicenseType: ptr.String("PARTNER")}, true},
		{"other license", internal.ERSContext{LicenseType: ptr.String("any other")}, false}} {
		t.Run(testCase.name, func(t *testing.T) {
			// when
			result := broker.IsExternalLicenseType(testCase.ersContext)
			// then
			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}

// assertions

func assertSecurityWithDefaultAdministrator(t *testing.T, runtime imv1.Runtime) {
	assert.ElementsMatch(t, runtime.Spec.Security.Administrators, []string{"User-operation-01"})
	assert.Equal(t, runtime.Spec.Security.Networking.Filter.Egress, imv1.Egress(imv1.Egress{Enabled: true}))
}

func assertSecurityEgressEnabled(t *testing.T, runtime imv1.Runtime) {
	assertSecurityWithEgressFilter(t, runtime, true)
}

func assertSecurityEgressDisabled(t *testing.T, runtime imv1.Runtime) {
	assertSecurityWithEgressFilter(t, runtime, false)
}

func assertSecurityWithEgressFilter(t *testing.T, runtime imv1.Runtime, egress bool) {
	assert.ElementsMatch(t, runtime.Spec.Security.Administrators, runtimeAdministrators)
	assert.Equal(t, imv1.Egress{Enabled: egress}, runtime.Spec.Security.Networking.Filter.Egress)
}

func assertSecurityWithNetworkFilter(t *testing.T, runtime imv1.Runtime, egress bool, ingress bool) {
	assert.ElementsMatch(t, runtime.Spec.Security.Administrators, runtimeAdministrators)
	assert.Equal(t, imv1.Egress{Enabled: egress}, runtime.Spec.Security.Networking.Filter.Egress)
	assert.Equal(t, &imv1.Ingress{Enabled: ingress}, runtime.Spec.Security.Networking.Filter.Ingress)
}

func assertLabelsKIMDriven(t *testing.T, preOperation internal.Operation, runtime imv1.Runtime) {
	assertLabels(t, preOperation, runtime)
}

func assertLabels(t *testing.T, operation internal.Operation, runtime imv1.Runtime) {
	assert.Equal(t, operation.InstanceID, runtime.Labels[customresources.InstanceIdLabel])
	assert.Equal(t, operation.RuntimeID, runtime.Labels[customresources.RuntimeIdLabel])
	assert.Equal(t, operation.ProvisioningParameters.PlanID, runtime.Labels[customresources.PlanIdLabel])
	assert.Equal(t, broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(operation.ProvisioningParameters.PlanID)), runtime.Labels[customresources.PlanNameLabel])
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.GlobalAccountID, runtime.Labels[customresources.GlobalAccountIdLabel])
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SubAccountID, runtime.Labels[customresources.SubaccountIdLabel])
	assert.Equal(t, operation.ShootName, runtime.Labels[customresources.ShootNameLabel])
	assert.Equal(t, *operation.ProvisioningParameters.Parameters.Region, runtime.Labels[customresources.RegionLabel])
	assert.Equal(t, operation.KymaResourceName, runtime.Labels[customresources.KymaNameLabel])
	if operation.ProvisioningParameters.PlatformRegion != "" {
		assert.Equal(t, operation.ProvisioningParameters.PlatformRegion, runtime.Labels[customresources.PlatformRegionLabel])
	}
}

func assertWorkers(t *testing.T, workers []gardener.Worker, machine string, maximum, minimum, maxSurge, maxUnavailable int, zoneCount int, zones []string) {
	assert.Len(t, workers, 1)
	assert.Len(t, workers[0].Zones, zoneCount)
	assert.Subset(t, zones, workers[0].Zones)
	assert.Equal(t, workers[0].Machine.Type, machine)
	assert.Equal(t, workers[0].MaxSurge.IntValue(), maxSurge)
	assert.Equal(t, workers[0].MaxUnavailable.IntValue(), maxUnavailable)
	assert.Equal(t, workers[0].Maximum, int32(maximum))
	assert.Equal(t, workers[0].Minimum, int32(minimum))
}

func assertWorkersWithVolume(t *testing.T, workers []gardener.Worker, machine string, maximum, minimum, maxSurge, maxUnavailable int, zoneCount int, zones []string, volumeSize, volumeType string) {
	assert.Len(t, workers, 1)
	assert.Len(t, workers[0].Zones, zoneCount)
	assert.Subset(t, zones, workers[0].Zones)
	assert.Equal(t, workers[0].Machine.Type, machine)
	assert.Equal(t, workers[0].MaxSurge.IntValue(), maxSurge)
	assert.Equal(t, workers[0].MaxUnavailable.IntValue(), maxUnavailable)
	assert.Equal(t, workers[0].Maximum, int32(maximum))
	assert.Equal(t, workers[0].Minimum, int32(minimum))
	assert.Equal(t, workers[0].Volume.VolumeSize, volumeSize)
	assert.Equal(t, *workers[0].Volume.Type, volumeType)
}

func assertNetworking(t *testing.T, expected imv1.Networking, actual imv1.Networking) {
	assert.True(t, reflect.DeepEqual(expected, actual))
}

func assertDefaultNetworking(t *testing.T, actual imv1.Networking) {
	assertNetworking(t, defaultNetworking, actual)
}

func assertInsertions(t *testing.T, memoryStorage storage.BrokerStorage, instance internal.Instance, operation internal.Operation) {
	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)
	err = memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)
}

// test fixtures

func getClientForTests(t *testing.T) client.Client {
	var cli client.Client
	if len(os.Getenv("KUBECONFIG")) > 0 && strings.ToLower(os.Getenv("USE_KUBECONFIG")) == "true" {
		config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			t.Fatal(err.Error())
		}

		cli, err = client.New(config, client.Options{})
		if err != nil {
			t.Fatal(err.Error())
		}
	} else {
		cli = fake.NewClientBuilder().Build()
	}
	return cli
}

func fixInstanceAndOperation(planID, region, platformRegion string, inputConfig broker.InfrastructureManager, platformProvider pkg.CloudProvider) (internal.Instance, internal.Operation) {
	instance := fixInstance()
	operation := fixOperationForCreateRuntimeResourceStep(OperationID, instance.InstanceID, planID, region, platformRegion, inputConfig, platformProvider)
	return instance, operation
}

func fixOperationForCreateRuntimeResourceStep(operationID, instanceID, planID, region, platformRegion string, inputConfig broker.InfrastructureManager, platformProvider pkg.CloudProvider) internal.Operation {
	var regionToSet *string
	if region != "" {
		regionToSet = &region

	}
	provisioningParameters := internal.ProvisioningParameters{
		PlanID:     planID,
		ServiceID:  fixture.ServiceId,
		ErsContext: fixture.FixERSContext(operationID),
		Parameters: pkg.ProvisioningParametersDTO{
			Name:                  "cluster-test",
			Region:                regionToSet,
			RuntimeAdministrators: runtimeAdministrators,
			TargetSecret:          ptr.String(SecretBindingName),
		},
		PlatformRegion: platformRegion,
	}

	operation := fixture.FixProvisioningOperation(operationID, instanceID, fixture.WithProvisioningParameters(provisioningParameters))
	operation.State = domain.InProgress
	operation.KymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
name: my-kyma
namespace: kyma-system
spec:
sync:
strategy: secret
channel: fast
modules: []
`
	operation.ProvisioningParameters.PlatformProvider = platformProvider

	planSpec, _ := provider.NewFakePlanSpecFromFile()
	valuesProvider := provider.NewPlanSpecificValuesProvider(inputConfig, nil, provider.FakeZonesProvider([]string{"a", "b", "c"}), planSpec)

	values, _ := valuesProvider.ValuesForPlanAndParameters(operation.ProvisioningParameters)
	operation.ProviderValues = &values
	return operation
}

func newTestProviderSpecWithDualStack() *configuration.ProviderSpec {
	// Create a test provider specification with dual-stack support for AWS and GCP, no support for Azure
	providerConfigYAML := `
aws:
  dualStack: true
  regions:
    eu-west-2:
      displayName: "Europe (London)"
      zones: ["a", "b", "c"]
azure:
  dualStack: false
  regions:
    westeurope:
      displayName: "West Europe"
      zones: ["1", "2", "3"]
gcp:
  dualStack: true
  regions:
    europe-west1:
      displayName: "Europe West 1"
      zones: ["a", "b", "c"]
`
	providerSpec, _ := configuration.NewProviderSpec(strings.NewReader(providerConfigYAML))
	return providerSpec
}
