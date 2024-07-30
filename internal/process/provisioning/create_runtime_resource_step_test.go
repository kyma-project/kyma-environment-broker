package provisioning

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

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
	"github.com/kyma-project/kyma-environment-broker/internal/kim"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

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
}

func TestCreateRuntimeResourceStep_Defaults_Azure_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("azure", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_Azure_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("azure", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_GCP_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.GCPPlanID, "asia-south1", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("gcp", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_GCP_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.GCPPlanID, "asia-south1", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("gcp", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("aws", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("aws", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_Preview_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("preview", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	postOperation, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	_, err = memoryStorage.Instances().GetByID(postOperation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("preview", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	postOperation, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	_, err = memoryStorage.Instances().GetByID(postOperation.InstanceID)
	assert.NoError(t, err)
}

// Actual creation tests

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("aws", false)
	inputConfig := input.Config{MultiZoneCluster: false, ControlPlaneFailureTolerance: "zone"}

	cli := getClientForTests(t)
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

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

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AWSPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("aws", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 3, 0, 3, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("preview", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_ActualCreation_WithRetry(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("preview", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	// then retry
	_, repeat, err = step.Run(operation, entry)
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
	assertSecurity(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 1, 0, 1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_Preview_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.PreviewPlanID, "eu-west-2", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("preview", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "aws", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "eu-west-2", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "m6i.large", 20, 3, 3, 0, 3, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_Azure_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("azure", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "azure", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "westeurope", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))

	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "Standard_D2s_v5", 20, 3, 1, 0, 1, []string{"1", "2", "3"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_Azure_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.AzurePlanID, "westeurope", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("azure", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "azure", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "westeurope", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))

	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "Standard_D2s_v5", 20, 3, 3, 0, 3, []string{"1", "2", "3"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_GCP_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.GCPPlanID, "asia-south1", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("gcp", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "gcp", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "asia-south1", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))

	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "n2-standard-2", 20, 3, 1, 0, 1, []string{"asia-south1-a", "asia-south1-b", "asia-south1-c"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_GCP_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance, operation := fixInstanceAndOperation(broker.GCPPlanID, "asia-south1", "platform-region")
	assertInsertions(t, memoryStorage, instance, operation)

	kimConfig := fixKimConfig("gcp", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(operation, entry)

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
	assertSecurity(t, runtime)

	assert.Equal(t, "gcp", runtime.Spec.Shoot.Provider.Type)
	assert.Equal(t, "asia-south1", runtime.Spec.Shoot.Region)
	assert.Equal(t, "production", string(runtime.Spec.Shoot.Purpose))
	assertWorkers(t, runtime.Spec.Shoot.Provider.Workers, "n2-standard-2", 20, 3, 3, 0, 3, []string{"asia-south1-a", "asia-south1-b", "asia-south1-c"})

	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

// assertions

func assertSecurity(t *testing.T, runtime imv1.Runtime) {
	assert.ElementsMatch(t, runtime.Spec.Security.Administrators, runtimeAdministrators)
	assert.Equal(t, runtime.Spec.Security.Networking.Filter.Egress, imv1.Egress(imv1.Egress{Enabled: false}))
}

func assertLabelsKIMDriven(t *testing.T, preOperation internal.Operation, runtime imv1.Runtime) {
	assertLabels(t, preOperation, runtime)

	provisionerDriven, ok := runtime.Labels["kyma-project.io/controlled-by-provisioner"]
	assert.True(t, !ok || provisionerDriven == "false")
}

func assertLabelsProvisionerDriven(t *testing.T, preOperation internal.Operation, runtime imv1.Runtime) {
	assertLabels(t, preOperation, runtime)

	provisionerDriven, ok := runtime.Labels["kyma-project.io/controlled-by-provisioner"]
	assert.True(t, ok && provisionerDriven == "true")
}

func assertLabels(t *testing.T, operation internal.Operation, runtime imv1.Runtime) {
	assert.Equal(t, operation.InstanceID, runtime.Labels["kyma-project.io/instance-id"])
	assert.Equal(t, operation.RuntimeID, runtime.Labels["kyma-project.io/runtime-id"])
	assert.Equal(t, operation.ProvisioningParameters.PlanID, runtime.Labels["kyma-project.io/broker-plan-id"])
	assert.Equal(t, broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID], runtime.Labels["kyma-project.io/broker-plan-name"])
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.GlobalAccountID, runtime.Labels["kyma-project.io/global-account-id"])
	assert.Equal(t, operation.ProvisioningParameters.ErsContext.SubAccountID, runtime.Labels["kyma-project.io/subaccount-id"])
	assert.Equal(t, operation.ShootName, runtime.Labels["kyma-project.io/shoot-name"])
	assert.Equal(t, *operation.ProvisioningParameters.Parameters.Region, runtime.Labels["kyma-project.io/region"])
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

func fixKimConfig(planName string, dryRun bool) kim.Config {
	return kim.Config{
		Enabled:  true,
		Plans:    []string{planName},
		ViewOnly: false,
		DryRun:   dryRun,
	}
}

func fixKimConfigProvisionerDriven(planName string, dryRun bool) kim.Config {
	return kim.Config{
		Enabled:  true,
		Plans:    []string{planName},
		ViewOnly: true,
		DryRun:   dryRun,
	}
}

func fixInstanceAndOperation(planID, region, platformRegion string) (internal.Instance, internal.Operation) {
	instance := fixInstance()
	operation := fixOperationForCreateRuntimeResourceStep(OperationID, instance.InstanceID, planID, region, platformRegion)
	return instance, operation
}

func fixOperationForCreateRuntimeResourceStep(operationID, instanceID, planID, region, platformRegion string) internal.Operation {
	provisioningParameters := internal.ProvisioningParameters{
		PlanID:     planID,
		ServiceID:  fixture.ServiceId,
		ErsContext: fixture.FixERSContext(operationID),
		Parameters: internal.ProvisioningParametersDTO{
			Name:                  "cluster-test",
			Region:                ptr.String(region),
			RuntimeAdministrators: runtimeAdministrators,
			TargetSecret:          ptr.String(SecretBindingName),
		},
		PlatformRegion: platformRegion,
	}

	operation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID, provisioningParameters)
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

func fixProvisionerParameters(cloudProvider internal.CloudProvider, region string) internal.ProvisioningParametersDTO {
	return internal.ProvisioningParametersDTO{
		Name:         "cluster-test",
		VolumeSizeGb: ptr.Integer(50),
		MachineType:  ptr.String("Standard_D8_v3"),
		Region:       ptr.String(region),
		Purpose:      ptr.String("Purpose"),
		LicenceType:  ptr.String("LicenceType"),
		Zones:        []string{"1"},
		AutoScalerParameters: internal.AutoScalerParameters{
			AutoScalerMin:  ptr.Integer(3),
			AutoScalerMax:  ptr.Integer(10),
			MaxSurge:       ptr.Integer(4),
			MaxUnavailable: ptr.Integer(1),
		},
		Provider: &cloudProvider,
	}
}
