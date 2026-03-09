package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
)

const (
	fixOperationID = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	fixInstanceID  = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
)

func TestInitStep_happyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	prepareProvisionedInstance(t, memoryStorage)
	dOp := prepareDeprovisioningOperation(t, memoryStorage, internal.OperationStatePending)

	svc := NewInitStep(memoryStorage, 90*time.Second)

	// when
	op, d, err := svc.Run(dOp, fixLogger())

	// then
	assert.Equal(t, domain.InProgress, op.State)
	assert.NoError(t, err)
	assert.Zero(t, d)
	dbOp, _ := memoryStorage.Operations().GetOperationByID(op.ID)
	assert.Equal(t, domain.InProgress, dbOp.State)
}

func TestInitStep_existingUpdatingOperation(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	prepareProvisionedInstance(t, memoryStorage)
	uOp := fixture.FixUpdatingOperation("uop-id", testInstanceID)
	uOp.State = domain.InProgress
	err := memoryStorage.Operations().InsertOperation(uOp.Operation)
	assert.NoError(t, err)
	dOp := prepareDeprovisioningOperation(t, memoryStorage, internal.OperationStatePending)

	svc := NewInitStep(memoryStorage, 90*time.Second)

	// when
	op, d, err := svc.Run(dOp, fixLogger())

	// then
	assert.Equal(t, internal.OperationStatePending, string(op.State))
	assert.NoError(t, err)
	assert.NotZero(t, d)
	dbOp, _ := memoryStorage.Operations().GetOperationByID(op.ID)
	assert.Equal(t, internal.OperationStatePending, string(dbOp.State))
}

func prepareProvisionedInstance(t *testing.T, s storage.BrokerStorage) {
	inst := fixture.FixInstance(testInstanceID)
	err := s.Instances().Insert(inst)
	assert.NoError(t, err)
	pOp := fixture.FixProvisioningOperation("pop-id", testInstanceID)
	err = s.Operations().InsertOperation(pOp)
	assert.NoError(t, err)
}

func prepareDeprovisioningOperation(t *testing.T, s storage.BrokerStorage, state domain.LastOperationState) internal.Operation {
	dOperation := fixture.FixDeprovisioningOperation("dop-id", testInstanceID)
	dOperation.State = state
	err := s.Operations().InsertOperation(dOperation.Operation)
	assert.NoError(t, err)
	return dOperation.Operation
}
