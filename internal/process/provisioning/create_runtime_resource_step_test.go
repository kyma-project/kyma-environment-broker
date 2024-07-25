package provisioning

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

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

var runtimeAdministrators = []string{"admin1@test.com", "admin2@test.com"}

func TestCreateRuntimeResourceStep_Defaults_Azure_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "westeurope"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.AzurePlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("azure", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_Azure_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "westeurope"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.AzurePlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("azure", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_GCP_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "asia-south1"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.GCPPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("gcp", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_GCP_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "asia-south1"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.GCPPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("gcp", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "eu-west-2"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.AWSPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("aws", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_SingleZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "eu-west-2"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.AWSPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("aws", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
}

func TestCreateRuntimeResourceStep_Defaults_Preview_MultiZone_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	region := "eu-west-2"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.PreviewPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("preview", true)
	inputConfig := input.Config{MultiZoneCluster: true}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	postOperation, repeat, err := step.Run(preOperation, entry)

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

	region := "eu-west-2"
	preOperation := fixture.FixProvisioningOperationWithProvisioningParameters(operationID, instanceID,
		fixture.FixProvisioningParametersWithDTO(operationID, broker.PreviewPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := fixKimConfig("preview", true)
	inputConfig := input.Config{MultiZoneCluster: false}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	postOperation, repeat, err := step.Run(preOperation, entry)

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

	region := "eu-west-2"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.AWSPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("aws", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "aws")
	assert.Equal(t, runtime.Spec.Shoot.Region, "eu-west-2")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 1) //TODO assert zone as an element from set

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_AWS_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "eu-west-2"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.AWSPlanID, fixProvisioningParametersDTOWithRegion(region)))
	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("aws", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "aws")
	assert.Equal(t, runtime.Spec.Shoot.Region, "eu-west-2")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 3) //TODO assert zones

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_Preview_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "eu-west-2"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.PreviewPlanID, fixProvisioningParametersDTOWithRegion(region)))
	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("preview", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "aws")
	assert.Equal(t, runtime.Spec.Shoot.Region, "eu-west-2")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 1) //TODO assert zone as an element from set

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_Preview_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "eu-west-2"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.PreviewPlanID, fixProvisioningParametersDTOWithRegion(region)))
	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("preview", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "aws")
	assert.Equal(t, runtime.Spec.Shoot.Region, "eu-west-2")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 3) //TODO assert zones

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)
}

func TestCreateRuntimeResourceStep_Defaults_Azure_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "westeurope"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.AzurePlanID, fixProvisioningParametersDTOWithRegion(region)))

	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("azure", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "azure")
	assert.Equal(t, runtime.Spec.Shoot.Region, "westeurope")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 1) //TODO assert zone as an element from set

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_Azure_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "westeurope"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.AzurePlanID, fixProvisioningParametersDTOWithRegion(region)))

	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("azure", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "azure")
	assert.Equal(t, runtime.Spec.Shoot.Region, "westeurope")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 3) //TODO assert zones

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_GCP_SingleZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "asia-south1"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.GCPPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("gcp", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: false}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "gcp")
	assert.Equal(t, runtime.Spec.Shoot.Region, "asia-south1")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 1) //TODO assert zone as an element from set

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_Defaults_GCP_MultiZone_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	region := "asia-south1"

	instance := fixInstance()
	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID, fixture.FixProvisioningParametersWithDTO(operationID, broker.GCPPlanID, fixProvisioningParametersDTOWithRegion(region)))

	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	kimConfig := fixKimConfig("gcp", false)

	cli := getClientForTests(t)
	inputConfig := input.Config{MultiZoneCluster: true}
	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), cli, kimConfig, inputConfig, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	runtime := imv1.Runtime{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: "kyma-system",
		Name:      preOperation.RuntimeID,
	}, &runtime)
	assert.NoError(t, err)
	assert.Equal(t, preOperation.RuntimeID, runtime.Name)
	assert.Equal(t, "runtime-58f8c703-1756-48ab-9299-a847974d1fee", runtime.Labels["operator.kyma-project.io/kyma-name"])

	assertLabels(t, preOperation, runtime)
	assertSecurity(t, runtime)

	assert.Equal(t, runtime.Spec.Shoot.Provider.Type, "gcp")
	assert.Equal(t, runtime.Spec.Shoot.Region, "asia-south1")
	assert.Equal(t, string(runtime.Spec.Shoot.Purpose), "production")
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers, 1)
	assert.Len(t, runtime.Spec.Shoot.Provider.Workers[0].Zones, 3) //TODO assert zones

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

// assertions and fixtures

func assertSecurity(t *testing.T, runtime imv1.Runtime) {
	assert.True(t, reflect.DeepEqual(runtime.Spec.Security.Administrators, runtimeAdministrators))
	assert.Equal(t, runtime.Spec.Security.Networking.Filter.Egress, imv1.Egress(imv1.Egress{Enabled: false}))
}

func assertLabels(t *testing.T, preOperation internal.Operation, runtime imv1.Runtime) {
	assert.Equal(t, preOperation.InstanceID, runtime.Labels["kyma-project.io/instance-id"])
	assert.Equal(t, preOperation.RuntimeID, runtime.Labels["kyma-project.io/runtime-id"])
	assert.Equal(t, preOperation.ProvisioningParameters.PlanID, runtime.Labels["kyma-project.io/broker-plan-id"])
	assert.Equal(t, broker.PlanNamesMapping[preOperation.ProvisioningParameters.PlanID], runtime.Labels["kyma-project.io/broker-plan-name"])
	assert.Equal(t, preOperation.ProvisioningParameters.ErsContext.GlobalAccountID, runtime.Labels["kyma-project.io/global-account-id"])
	assert.Equal(t, preOperation.ProvisioningParameters.ErsContext.SubAccountID, runtime.Labels["kyma-project.io/subaccount-id"])
	assert.Equal(t, preOperation.ShootName, runtime.Labels["kyma-project.io/shoot-name"])
	assert.Equal(t, *preOperation.ProvisioningParameters.Parameters.Region, runtime.Labels["kyma-project.io/region"])
}

func fixOperationForCreateRuntimeResource(instanceID string, provisioningParameters internal.ProvisioningParameters) internal.Operation {
	operation := fixture.FixProvisioningOperationWithProvisioningParameters("op-id", instanceID, provisioningParameters)
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

func fixProvisioningParametersDTOWithRegion(region string) internal.ProvisioningParametersDTO {
	return internal.ProvisioningParametersDTO{
		Name:                  "cluster-test",
		Region:                ptr.String(region),
		RuntimeAdministrators: runtimeAdministrators,
	}
}
