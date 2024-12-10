package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func TestStartStep_RunIfDeprovisioningInProgress(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	step := NewStartStep(st.Operations(), st.Instances())
	dOp := fixture.FixDeprovisioningOperation("d-op-id", "instance-id")
	dOp.State = domain.InProgress
	dOp.Temporary = true
	pOp := fixture.FixProvisioningOperation("p-op-id", "instance-id")
	pOp.State = orchestration.Pending
	inst := fixture.FixInstance("instance-id")

	err := st.Instances().Insert(inst)
	assert.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(dOp)
	assert.NoError(t, err)
	err = st.Operations().InsertOperation(pOp)
	assert.NoError(t, err)

	// when
	operation, retry, err := step.Run(pOp, fixLogger())

	// then
	assert.Equal(t, domain.LastOperationState(orchestration.Pending), operation.State)
	assert.NoError(t, err)
	assert.NotZero(t, retry)
}

func TestStartStep_RunIfDeprovisioningDone(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	step := NewStartStep(st.Operations(), st.Instances())
	dOp := fixture.FixDeprovisioningOperation("d-op-id", "instance-id")
	dOp.State = domain.Succeeded
	dOp.Temporary = true
	pOp := fixture.FixProvisioningOperation("p-op-id", "instance-id")
	pOp.State = orchestration.Pending
	inst := fixture.FixInstance("instance-id")

	err := st.Instances().Insert(inst)
	assert.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(dOp)
	assert.NoError(t, err)
	err = st.Operations().InsertOperation(pOp)
	assert.NoError(t, err)

	// when
	operation, retry, err := step.Run(pOp, fixLogger())

	// then
	assert.Equal(t, domain.InProgress, operation.State)
	assert.NoError(t, err)
	assert.Zero(t, retry)
	storedOp, err := st.Operations().GetOperationByID("p-op-id")
	assert.NoError(t, err)
	assert.Equal(t, domain.InProgress, storedOp.State)
}
