package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/kim"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateRuntimeResourceStep_HappyPath_YamlOnly(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	preOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	preOperation.KymaTemplate = `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
    name: gophers-test-kyma
    namespace: kyma-system
spec:
    sync:
        strategy: secret
    channel: stable
    modules: []
`
	preOperation.KymaResourceNamespace = "kyma-system"
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

	step := NewCreateRuntimeResourceStep(memoryStorage.Operations(), memoryStorage.RuntimeStates(), memoryStorage.Instances(), kimConfig)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}
