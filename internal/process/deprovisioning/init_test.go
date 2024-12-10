package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

const (
	fixOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	fixInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	fixRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	fixGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	fixProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"
)

func TestInitStep_happyPath(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	prepareProvisionedInstance(t, memoryStorage)
	dOp := prepareDeprovisioningOperation(t, memoryStorage, orchestration.Pending)

	svc := NewInitStep(memoryStorage.Operations(), memoryStorage.Instances(), 90*time.Second)

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
	dOp := prepareDeprovisioningOperation(t, memoryStorage, orchestration.Pending)

	svc := NewInitStep(memoryStorage.Operations(), memoryStorage.Instances(), 90*time.Second)

	// when
	op, d, err := svc.Run(dOp, fixLogger())

	// then
	assert.Equal(t, orchestration.Pending, string(op.State))
	assert.NoError(t, err)
	assert.NotZero(t, d)
	dbOp, _ := memoryStorage.Operations().GetOperationByID(op.ID)
	assert.Equal(t, orchestration.Pending, string(dbOp.State))
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

func fixDeprovisioningOperation() internal.DeprovisioningOperation {
	deprovisioniningOperation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
	return deprovisioniningOperation
}

func fixProvisioningOperation() internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperation(fixOperationID, fixInstanceID)
	return provisioningOperation
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(fixInstanceID)
	instance.RuntimeID = fixRuntimeID
	instance.GlobalAccountID = fixGlobalAccountID

	return instance
}
