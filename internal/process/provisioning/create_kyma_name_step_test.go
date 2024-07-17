package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateKymaNameStep_HappyPath(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	preOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	err := memoryStorage.Operations().InsertOperation(preOperation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateKymaNameStep(memoryStorage.Operations())

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, repeat, err := step.Run(preOperation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)

	_, err = memoryStorage.Instances().GetByID(preOperation.InstanceID)
	assert.NoError(t, err)

}
