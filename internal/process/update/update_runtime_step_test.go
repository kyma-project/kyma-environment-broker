package update

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/process/infrastructure_manager"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/require"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateRuntimeStep_NoRuntime(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	db := storage.NewMemoryStorage()
	operations := db.Operations()

	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kyma-ns"
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	step := NewUpdateRuntimeStep(operations, kcpClient, 0, infrastructure_manager.InfrastructureManagerConfig{}, false, nil)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.Error(t, err)
}

func TestUpdateRuntimeStep_RunUpdateMachineType(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name", false)).Build()
	step := NewUpdateRuntimeStep(nil, kcpClient, 0, infrastructure_manager.InfrastructureManagerConfig{}, false, nil)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	operation.UpdatingParameters = internal.UpdatingParametersDTO{
		MachineType: ptr.String("new-machine-type"),
	}

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)

	var gotRuntime imv1.Runtime
	err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &gotRuntime)
	require.NoError(t, err)
	assert.Equal(t, "new-machine-type", gotRuntime.Spec.Shoot.Provider.Workers[0].Machine.Type)

}

func TestUpdateRuntimeStep_RunUpdateOnlyMainOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name", false)).Build()
	step := NewUpdateRuntimeStep(nil, kcpClient, 0, infrastructure_manager.InfrastructureManagerConfig{
		UseMainOIDC:       true,
		UseAdditionalOIDC: false,
	}, false, nil)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	expectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("clinet-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("issuer-url"),
		SigningAlgs:    []string{"signingAlgs"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: nil,
	}

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)

	var gotRuntime imv1.Runtime
	err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &gotRuntime)
	require.NoError(t, err)
	assert.Equal(t, expectedOIDCConfig, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
}

func TestUpdateRuntimeStep_RunUpdateMainAndAdditionalOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name", false)).Build()
	step := NewUpdateRuntimeStep(nil, kcpClient, 0, infrastructure_manager.InfrastructureManagerConfig{UseMainOIDC: true,
		UseAdditionalOIDC: true,
	}, false, nil)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	expectedMainOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("clinet-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("issuer-url"),
		SigningAlgs:    []string{"signingAlgs"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: nil,
	}
	expectedAdditionalOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("clinet-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("issuer-url"),
		SigningAlgs:    []string{"signingAlgs"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: nil,
		GroupsPrefix:   ptr.String("-"),
	}

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)

	var gotRuntime imv1.Runtime
	err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &gotRuntime)
	require.NoError(t, err)
	assert.Equal(t, expectedMainOIDCConfig, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig)
	assert.NotNil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
	assert.Equal(t, expectedAdditionalOIDCConfig, (*gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func TestUpdateRuntimeStep_RunUpdateOnlyAdditionalOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name", false)).Build()
	step := NewUpdateRuntimeStep(nil, kcpClient, 0, infrastructure_manager.InfrastructureManagerConfig{
		UseMainOIDC:       false,
		UseAdditionalOIDC: true,
	}, false, nil)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	expectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("clinet-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("issuer-url"),
		SigningAlgs:    []string{"signingAlgs"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: nil,
		GroupsPrefix:   ptr.String("-"),
	}
	var gotRuntime imv1.Runtime
	err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &gotRuntime)
	require.NoError(t, err)
	t.Logf("gotRuntime: %+v", gotRuntime)
	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)

	err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &gotRuntime)
	require.NoError(t, err)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim)
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix)
	assert.NotNil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
	assert.Equal(t, expectedOIDCConfig, (*gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
}

func fixRuntimeResource(name string, controlledByProvisioner bool) runtime.Object {
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
			Labels: map[string]string{
				imv1.LabelControlledByProvisioner: strconv.FormatBool(controlledByProvisioner),
			},
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Provider: imv1.Provider{
					Workers: []gardener.Worker{
						{
							Machine: gardener.Machine{
								Type: "original-type",
							},
							MaxSurge:       &maxSurge,
							MaxUnavailable: &maxUnavailable,
						},
					},
				},
			},
		},
	}
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("testing", true)
}
