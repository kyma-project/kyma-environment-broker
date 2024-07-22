package provisioning

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

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

var expectedAdministrators = []string{"admin1@test.com", "admin2@test.com"}

func TestCreateRuntimeResourceStep_HappyPath_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	preOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	kimConfig := kim.Config{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: false,
		DryRun:   true,
	}

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, kimConfig, input.Config{}, nil, false)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

func TestCreateRuntimeResourceStep_HappyPath_ActualCreation(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	err := imv1.AddToScheme(scheme.Scheme)

	instance := fixInstance()
	preOperation := fixOperationForCreateRuntimeResource(instance.InstanceID)

	err = memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	kimConfig := kim.Config{
		Enabled:  true,
		Plans:    []string{"azure"},
		ViewOnly: false,
		DryRun:   false,
	}

	cli := getClientForTests(t)
	inputConfig := input.Config{}
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

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}

func assertSecurity(t *testing.T, runtime imv1.Runtime) {
	assert.True(t, reflect.DeepEqual(runtime.Spec.Security.Administrators, expectedAdministrators))
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

func fixOperationForCreateRuntimeResource(instanceID string) internal.Operation {
	operation := fixture.FixOperation("op-id", instanceID, internal.OperationTypeProvision)

	operation.ProvisioningParameters.Parameters.RuntimeAdministrators = expectedAdministrators
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
