package update

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var memoryStorage = storage.NewMemoryStorage()

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

	step := NewUpdateRuntimeStep(db, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})

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
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name")).Build()
	step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})
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

func TestUpdateRuntimeStep_RunUpdateEmptyOIDCConfigWithOIDCObject(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name")).Build()
	step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})
	operation := fixture.FixUpdatingOperationWithOIDCObject("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	expectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("client-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("issuer-url"),
		SigningAlgs:    []string{"signingAlgs"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: nil,
		RequiredClaims: map[string]string{
			"claim1": "value1",
			"claim2": "value2",
		},
		GroupsPrefix: ptr.String("-"),
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

func TestUpdateRuntimeStep_RunUpdateOIDCWithOIDCObject(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResourceWithOneAdditionalOidc("runtime-name")).Build()
	step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})
	operation := fixture.FixUpdatingOperationWithOIDCObject("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	expectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("client-id-oidc"),
		GroupsClaim:    ptr.String("groups"),
		IssuerURL:      ptr.String("issuer-url"),
		SigningAlgs:    []string{"signingAlgs"},
		UsernameClaim:  ptr.String("sub"),
		UsernamePrefix: ptr.String("initial-username-prefix"),
		RequiredClaims: map[string]string{
			"claim1": "value1",
			"claim2": "value2",
		},
		GroupsPrefix: ptr.String("-"),
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

func TestUpdateRuntimeStep_RunUpdateEmptyAdditionalOIDCWithMultipleAdditionalOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name")).Build()
	step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	operation.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			List: []pkg.OIDCConfigDTO{
				{
					ClientID:       "first-client-id-custom",
					GroupsClaim:    "first-gc-custom",
					GroupsPrefix:   "first-gp-custom",
					IssuerURL:      "first-issuer-url-custom",
					SigningAlgs:    []string{"first-sa-custom"},
					UsernameClaim:  "first-uc-custom",
					UsernamePrefix: "first-up-custom",
					RequiredClaims: []string{"claim1=value1", "claim2=value2"},
				},
				{
					ClientID:       "second-client-id-custom",
					GroupsClaim:    "second-gc-custom",
					GroupsPrefix:   "second-gp-custom",
					IssuerURL:      "second-issuer-url-custom",
					SigningAlgs:    []string{"second-sa-custom"},
					UsernameClaim:  "second-uc-custom",
					UsernamePrefix: "second-up-custom",
				},
			},
		},
	}
	firstExpectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("first-client-id-custom"),
		GroupsClaim:    ptr.String("first-gc-custom"),
		IssuerURL:      ptr.String("first-issuer-url-custom"),
		SigningAlgs:    []string{"first-sa-custom"},
		UsernameClaim:  ptr.String("first-uc-custom"),
		UsernamePrefix: ptr.String("first-up-custom"),
		RequiredClaims: map[string]string{
			"claim1": "value1",
			"claim2": "value2",
		},
		GroupsPrefix: ptr.String("first-gp-custom"),
	}
	secondExpectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("second-client-id-custom"),
		GroupsClaim:    ptr.String("second-gc-custom"),
		IssuerURL:      ptr.String("second-issuer-url-custom"),
		SigningAlgs:    []string{"second-sa-custom"},
		UsernameClaim:  ptr.String("second-uc-custom"),
		UsernamePrefix: ptr.String("second-up-custom"),
		GroupsPrefix:   ptr.String("second-gp-custom"),
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
	assert.Equal(t, firstExpectedOIDCConfig, (*gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
	assert.Equal(t, secondExpectedOIDCConfig, (*gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[1])
}

func TestUpdateRuntimeStep_RunUpdateMultipleAdditionalOIDCWithMultipleAdditionalOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResourceWithMultipleAdditionalOidc("runtime-name")).Build()
	step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	operation.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			List: []pkg.OIDCConfigDTO{
				{
					ClientID:       "first-client-id-custom",
					GroupsClaim:    "first-gc-custom",
					GroupsPrefix:   "first-gp-custom",
					IssuerURL:      "first-issuer-url-custom",
					SigningAlgs:    []string{"first-sa-custom"},
					UsernameClaim:  "first-uc-custom",
					UsernamePrefix: "first-up-custom",
					RequiredClaims: []string{"claim1=value1", "claim2=value2"},
				},
				{
					ClientID:       "second-client-id-custom",
					GroupsClaim:    "second-gc-custom",
					GroupsPrefix:   "second-gp-custom",
					IssuerURL:      "second-issuer-url-custom",
					SigningAlgs:    []string{"second-sa-custom"},
					UsernameClaim:  "second-uc-custom",
					UsernamePrefix: "second-up-custom",
					RequiredClaims: []string{"claim3=value3", "claim4=value4"},
				},
			},
		},
	}
	firstExpectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("first-client-id-custom"),
		GroupsClaim:    ptr.String("first-gc-custom"),
		IssuerURL:      ptr.String("first-issuer-url-custom"),
		SigningAlgs:    []string{"first-sa-custom"},
		UsernameClaim:  ptr.String("first-uc-custom"),
		UsernamePrefix: ptr.String("first-up-custom"),
		RequiredClaims: map[string]string{
			"claim1": "value1",
			"claim2": "value2",
		},
		GroupsPrefix: ptr.String("first-gp-custom"),
	}
	secondExpectedOIDCConfig := gardener.OIDCConfig{
		ClientID:       ptr.String("second-client-id-custom"),
		GroupsClaim:    ptr.String("second-gc-custom"),
		IssuerURL:      ptr.String("second-issuer-url-custom"),
		SigningAlgs:    []string{"second-sa-custom"},
		UsernameClaim:  ptr.String("second-uc-custom"),
		UsernamePrefix: ptr.String("second-up-custom"),
		RequiredClaims: map[string]string{
			"claim3": "value3",
			"claim4": "value4",
		},
		GroupsPrefix: ptr.String("second-gp-custom"),
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
	assert.Nil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.RequiredClaims)
	assert.NotNil(t, gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)
	assert.Len(t, *gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 2)
	assert.Equal(t, firstExpectedOIDCConfig, (*gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0])
	assert.Equal(t, secondExpectedOIDCConfig, (*gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[1])
}

func TestUpdateRuntimeStep_RunUpdateMultipleAdditionalOIDCWitEmptyAdditionalOIDC(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResourceWithMultipleAdditionalOidc("runtime-name")).Build()
	step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, broker.InfrastructureManager{}, nil, true, &workers.Provider{})
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	operation.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			List: []pkg.OIDCConfigDTO{},
		},
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
	assert.Len(t, *gotRuntime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig, 0)
}

func TestUpdateRuntimeStep_NetworkFilter(t *testing.T) {
	// given
	for _, testCase := range []struct {
		name string

		initialEgressFiltering  bool
		initialIngressFiltering bool

		ingressFilteringFlag      bool
		planID                    string
		licenseType               string
		ingressFilteringParameter *bool

		expectedEgressResult  bool
		expectedIngressResult bool
	}{
		// legacy behavior
		{"Feature flag off - external", true, true, false, broker.SapConvergedCloudPlanID, "CUSTOMER", ptr.Bool(true), false, true},
		{"Feature flag off - internal", false, true, false, broker.SapConvergedCloudPlanID, "NON-CUSTOMER", ptr.Bool(true), true, true},
		{"Feature flag off - internal", false, false, false, broker.SapConvergedCloudPlanID, "NON-CUSTOMER", ptr.Bool(true), true, false},

		// new behavior - external account and no parameter - not updating ingress at all
		{"External- SapConvergedCloud - no parameter", true, true, true, broker.SapConvergedCloudPlanID, "CUSTOMER", nil, false, true},
		{"External- SapConvergedCloud - no parameter", true, false, true, broker.SapConvergedCloudPlanID, "CUSTOMER", nil, false, false},
		{"External - AWS", true, true, true, broker.AWSPlanID, "CUSTOMER", nil, false, true},
		{"External - AWS", true, false, true, broker.AWSPlanID, "CUSTOMER", nil, false, false},

		// new behavior - internal
		{"Internal - AWS - no parameter", true, true, true, broker.AWSPlanID, "NON-CUSTOMER", nil, true, true},
		{"Internal - AWS - turn on", true, true, true, broker.AWSPlanID, "NON-CUSTOMER", ptr.Bool(true), true, true},
		{"Internal - AWS - turn off", true, true, true, broker.AWSPlanID, "NON-CUSTOMER", ptr.Bool(false), true, false},
		{"Internal - AWS - no parameter", false, false, true, broker.AWSPlanID, "NON-CUSTOMER", nil, true, false},
		{"Internal - AWS - turn on ingress", false, false, true, broker.AWSPlanID, "NON-CUSTOMER", ptr.Bool(true), true, true},
		{"Internal - AWS - turn off ingress", false, false, true, broker.AWSPlanID, "NON-CUSTOMER", ptr.Bool(false), true, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// when
			err := imv1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)

			inputConfig := broker.InfrastructureManager{EnableIngressFiltering: testCase.ingressFilteringFlag,
				MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction,
				IngressFilteringPlans: []string{"aws", "gcp", "azure"}}

			operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
			operation.RuntimeResourceName = "runtime-name"
			operation.KymaResourceNamespace = "kcp-system"
			operation.UpdatingParameters = internal.UpdatingParametersDTO{
				IngressFiltering: testCase.ingressFilteringParameter,
			}

			operation.ProvisioningParameters.ErsContext.LicenseType = ptr.String(testCase.licenseType)
			operation.ProvisioningParameters.Parameters.IngressFiltering = testCase.ingressFilteringParameter

			kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResourceWithNetworkFilter("runtime-name", testCase.initialIngressFiltering, testCase.initialEgressFiltering)).Build()
			step := NewUpdateRuntimeStep(memoryStorage, kcpClient, 0, inputConfig, nil, true, &workers.Provider{})

			// when
			_, backoff, err := step.Run(operation, fixLogger())

			// then
			assert.NoError(t, err)
			assert.Zero(t, backoff)

			runtime := imv1.Runtime{}
			err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &runtime)
			require.NoError(t, err)

			assert.Equal(t, imv1.Egress{Enabled: testCase.expectedEgressResult}, runtime.Spec.Security.Networking.Filter.Egress)
			assert.Equal(t, &imv1.Ingress{Enabled: testCase.expectedIngressResult}, runtime.Spec.Security.Networking.Filter.Ingress)

		})
	}
}

// fixtures

func fixRuntimeResource(name string) runtime.Object {
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
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

func fixRuntimeResourceWithNetworkFilter(name string, ingressFilter, egressFilter bool) runtime.Object {
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
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
			Security: imv1.Security{
				Networking: imv1.NetworkingSecurity{
					Filter: imv1.Filter{
						Ingress: &imv1.Ingress{Enabled: ingressFilter},
						Egress:  imv1.Egress{Enabled: egressFilter},
					},
				},
			},
		},
	}
}

func fixRuntimeResourceWithOneAdditionalOidc(name string) runtime.Object {
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Kubernetes: imv1.Kubernetes{
					KubeAPIServer: imv1.APIServer{
						AdditionalOidcConfig: &[]gardener.OIDCConfig{
							{
								ClientID:       ptr.String("initial-client-id-oidc"),
								GroupsClaim:    ptr.String("initial-groups"),
								GroupsPrefix:   ptr.String("initial-groups-prefix"),
								IssuerURL:      ptr.String("initial-issuer-url"),
								SigningAlgs:    []string{"initial-signingAlgs"},
								UsernameClaim:  ptr.String("initial-sub"),
								UsernamePrefix: ptr.String("initial-username-prefix"),
							},
						},
					},
				},
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

func fixRuntimeResourceWithMultipleAdditionalOidc(name string) runtime.Object {
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Kubernetes: imv1.Kubernetes{
					KubeAPIServer: imv1.APIServer{
						AdditionalOidcConfig: &[]gardener.OIDCConfig{
							{
								ClientID:       ptr.String("first-initial-client-id-oidc"),
								GroupsClaim:    ptr.String("first-initial-groups"),
								GroupsPrefix:   ptr.String("first-initial-groups-prefix"),
								IssuerURL:      ptr.String("first-initial-issuer-url"),
								SigningAlgs:    []string{"first-initial-signingAlgs"},
								UsernameClaim:  ptr.String("first-initial-sub"),
								UsernamePrefix: ptr.String("first-initial-username-prefix"),
							},
							{
								ClientID:       ptr.String("second-initial-client-id-oidc"),
								GroupsClaim:    ptr.String("second-initial-groups"),
								GroupsPrefix:   ptr.String("second-initial-groups-prefix"),
								IssuerURL:      ptr.String("second-initial-issuer-url"),
								SigningAlgs:    []string{"second-initial-signingAlgs"},
								UsernameClaim:  ptr.String("second-initial-sub"),
								UsernamePrefix: ptr.String("second-initial-username-prefix"),
							},
							{
								ClientID:       ptr.String("third-initial-client-id-oidc"),
								GroupsClaim:    ptr.String("third-initial-groups"),
								GroupsPrefix:   ptr.String("third-initial-groups-prefix"),
								IssuerURL:      ptr.String("third-initial-issuer-url"),
								SigningAlgs:    []string{"third-initial-signingAlgs"},
								UsernameClaim:  ptr.String("third-initial-sub"),
								UsernamePrefix: ptr.String("third-initial-username-prefix"),
							},
						},
					},
				},
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
