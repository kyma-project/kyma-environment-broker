package provisioning

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/customresources"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	"github.com/pivotal-cf/brokerapi/v12/domain"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/process/input"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"k8s.io/client-go/kubernetes/scheme"
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
	//TODO: remove after KIM is handling this properly
	Type: ptr.String("calico"),
}

var defaultOIDSConfig = pkg.OIDCConfigDTO{
	ClientID:       "client-id-default",
	GroupsClaim:    "gc-default",
	IssuerURL:      "issuer-url-default",
	SigningAlgs:    []string{"sa-default"},
	UsernameClaim:  "uc-default",
	UsernamePrefix: "up-default",
}

func TestCreateRuntimeResourceStep_OIDC_AllCustom(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConfigDTO{
		ClientID:       "client-id-custom",
		GroupsClaim:    "gc-custom",
		IssuerURL:      "issuer-url-custom",
		SigningAlgs:    []string{"sa-custom"},
		UsernameClaim:  "uc-custom",
		UsernamePrefix: "up-custom",
	}
	assertInsertions(t, memoryStorage, instance, operation)
	inputConfig := input.Config{MultiZoneCluster: true}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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
	assert.Equal(t, gardener.OIDCConfig{
		ClientID:       ptr.String("client-id-custom"),
		GroupsClaim:    ptr.String("gc-custom"),
		IssuerURL:      ptr.String("issuer-url-custom"),
		SigningAlgs:    []string{"sa-custom"},
		UsernameClaim:  ptr.String("uc-custom"),
		UsernamePrefix: ptr.String("up-custom"),
	}, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig)
}

func TestCreateRuntimeResourceStep_OIDC_MixedCustom(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	memoryStorage := storage.NewMemoryStorage()
	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	operation.ProvisioningParameters.Parameters.OIDC = &pkg.OIDCConfigDTO{
		ClientID:      "client-id-custom",
		GroupsClaim:   "gc-custom",
		IssuerURL:     "issuer-url-custom",
		UsernameClaim: "uc-custom",
	}
	assertInsertions(t, memoryStorage, instance, operation)
	inputConfig := input.Config{MultiZoneCluster: true}
	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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
	assert.Equal(t, gardener.OIDCConfig{
		ClientID:       ptr.String("client-id-custom"),
		GroupsClaim:    ptr.String("gc-custom"),
		IssuerURL:      ptr.String("issuer-url-custom"),
		SigningAlgs:    []string{"sa-default"},
		UsernameClaim:  ptr.String("uc-custom"),
		UsernamePrefix: ptr.String("up-default"),
	}, runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig)
}

func TestCreateRuntimeResourceStep_FailureToleranceForTrial(t *testing.T) {
	// given
	assert.NoError(t, imv1.AddToScheme(scheme.Scheme))
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.TrialPlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = "zone"
	inputConfig.DefaultTrialProvider = "AWS"

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = "zone"

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = ""

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: true}
	inputConfig.ControlPlaneFailureTolerance = "node"

	cli := getClientForTests(t)

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_EnforceSeed_ActualCreation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	operation.ProvisioningParameters.Parameters.ShootAndSeedSameRegion = ptr.Bool(true)
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_DisableEnterpriseFilter_ActualCreation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	operation.ProvisioningParameters.ErsContext.LicenseType = ptr.String("PARTNER")
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_DefaultAdmin_ActualCreation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	operation.ProvisioningParameters.Parameters.RuntimeAdministrators = nil
	assertInsertions(t, memoryStorage, instance, operation)

	inputConfig := input.Config{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone", DefaultGardenerShootPurpose: provider.PurposeProduction}

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZoneWithNetworking_ActualCreation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	operation.ProvisioningParameters.Parameters.Networking = &pkg.NetworkingDTO{
		NodesCidr:    "192.168.48.0/20",
		PodsCidr:     ptr.String("10.104.0.0/24"),
		ServicesCidr: ptr.String("10.105.0.0/24"),
	}

	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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
		//TODO remove after KIM is handling this properly
		Type: ptr.String("calico"),
	}, runtime.Spec.Shoot.Networking)

	assert.Equal(t, "any-string", string(runtime.Spec.Shoot.ControlPlane.HighAvailability.FailureTolerance.Type))

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZone_ActualCreation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "any-string"}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_ActualCreation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "zone"}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_ActualCreation_WithRetry(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false, DefaultGardenerShootPurpose: provider.PurposeProduction, ControlPlaneFailureTolerance: "zone"}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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
		{"Single zone", pkg.SapConvergedCloud, 1, "openstack", "g_c2_m8", "eu-de-1", []string{"eu-de-1a", "eu-de-1b", "eu-de-1d"}},
		{"Multi zone", pkg.SapConvergedCloud, 3, "openstack", "g_c2_m8", "eu-de-1", []string{"eu-de-1a", "eu-de-1b", "eu-de-1d"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			memoryStorage := storage.NewMemoryStorage()
			err := imv1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)
			instance, operation := fixInstanceAndOperation(broker.SapConvergedCloudPlanID, "", "platform-region")
			operation.ProvisioningParameters.PlatformProvider = testCase.gotProvider
			assertInsertions(t, memoryStorage, instance, operation)

			cli := getClientForTests(t)
			inputConfig := input.Config{MultiZoneCluster: testCase.expectedZonesCount > 1, ControlPlaneFailureTolerance: "zone"}
			step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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
		{"azure", pkg.Azure, "azure", "Standard_D4s_v5", "westeurope", []string{"1", "2", "3"}},
		{"aws", pkg.AWS, "aws", "m5.xlarge", "westeurope", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			memoryStorage := storage.NewMemoryStorage()
			err := imv1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)
			instance, operation := fixInstanceAndOperation(broker.FreemiumPlanID, "", "platform-region")
			operation.ProvisioningParameters.PlatformProvider = testCase.gotProvider
			assertInsertions(t, memoryStorage, instance, operation)

			cli := getClientForTests(t)
			inputConfig := input.Config{MultiZoneCluster: true}
			step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, inputConfig, nil, false, defaultOIDSConfig)

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

// testing auxiliary functions

func Test_Defaults(t *testing.T) {
	//given
	//when

	nilToDefaultString := DefaultIfParamNotSet("default value", nil)
	nonDefaultString := DefaultIfParamNotSet("default value", ptr.String("initial value"))

	nilToDefaultInt := DefaultIfParamNotSet(42, nil)
	nonDefaultInt := DefaultIfParamNotSet(42, ptr.Integer(7))

	//then
	assert.Equal(t, "initial value", nonDefaultString)
	assert.Equal(t, "default value", nilToDefaultString)
	assert.Equal(t, 42, nilToDefaultInt)
	assert.Equal(t, 7, nonDefaultInt)
}

// assertions

func assertSecurityWithDefaultAdministrator(t *testing.T, runtime imv1.Runtime) {
	assert.ElementsMatch(t, runtime.Spec.Security.Administrators, []string{"User-operation-01"})
	assert.Equal(t, runtime.Spec.Security.Networking.Filter.Egress, imv1.Egress(imv1.Egress{Enabled: true}))
}

func assertSecurityEgressEnabled(t *testing.T, runtime imv1.Runtime) {
	assertSecurityWithNetworkingFilter(t, runtime, true)
}

func assertSecurityEgressDisabled(t *testing.T, runtime imv1.Runtime) {
	assertSecurityWithNetworkingFilter(t, runtime, false)
}

func assertSecurityWithNetworkingFilter(t *testing.T, runtime imv1.Runtime, egress bool) {
	assert.ElementsMatch(t, runtime.Spec.Security.Administrators, runtimeAdministrators)
	assert.Equal(t, runtime.Spec.Security.Networking.Filter.Egress, imv1.Egress{Enabled: egress})
}

func assertLabelsKIMDriven(t *testing.T, preOperation internal.Operation, runtime imv1.Runtime) {
	assertLabels(t, preOperation, runtime)
}

func assertLabels(t *testing.T, operation internal.Operation, runtime imv1.Runtime) {
	assert.Equal(t, operation.InstanceID, runtime.Labels[customresources.InstanceIdLabel])
	assert.Equal(t, operation.RuntimeID, runtime.Labels[customresources.RuntimeIdLabel])
	assert.Equal(t, operation.ProvisioningParameters.PlanID, runtime.Labels[customresources.PlanIdLabel])
	assert.Equal(t, broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID], runtime.Labels[customresources.PlanNameLabel])
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
		fmt.Println("using kubeconfig")
	} else {
		fmt.Println("using fake client")
		cli = fake.NewClientBuilder().Build()
	}
	return cli
}

func fixInstanceAndOperation(planID, region, platformRegion string) (internal.Instance, internal.Operation) {
	instance := fixInstance()
	operation := fixOperationForCreateRuntimeResourceStep(OperationID, instance.InstanceID, planID, region, platformRegion)
	return instance, operation
}

func fixOperationForCreateRuntimeResourceStep(operationID, instanceID, planID, region, platformRegion string) internal.Operation {
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

	operation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID, provisioningParameters)
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
channel: stable
modules: []
`
	return operation
}
