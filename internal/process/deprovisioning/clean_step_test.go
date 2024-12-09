package deprovisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/stretchr/testify/assert"
)

func TestCleanStep_Run(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	provisioning := fixture.FixProvisioningOperation("prov-id", "inst-id")
	deprovisioning := fixture.FixDeprovisioningOperationAsOperation("deprov-id", "inst-id")
	rs := fixture.FixRuntimeState("rs", provisioning.RuntimeID, "prov-id")
	err := db.Operations().InsertOperation(provisioning)
	assert.NoError(t, err)
	err = db.Operations().InsertOperation(deprovisioning)
	assert.NoError(t, err)
	err = db.RuntimeStates().Insert(rs)
	assert.NoError(t, err)

	step := NewCleanStep(db.Operations(), db.RuntimeStates(), false)

	// when
	_, backoff, err := step.Run(deprovisioning, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
	ops, err := db.Operations().ListOperationsByInstanceID("inst-id")
	assert.NoError(t, err)
	assert.Emptyf(t, ops, "Operations should be empty")
	runtimeStates, err := db.RuntimeStates().GetByOperationID("prov-id")
	assert.True(t, dberr.IsNotFound(err))
	assert.Emptyf(t, runtimeStates, "Runtime states should be empty")
}

func TestCleanStep_Run_TemporaryOperation(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	provisioning := fixture.FixProvisioningOperation("prov-id", "inst-id")
	deprovisioning := fixture.FixDeprovisioningOperationAsOperation("deprov-id", "inst-id")
	deprovisioning.Temporary = true

	err := db.Operations().InsertOperation(provisioning)
	assert.NoError(t, err)
	err = db.Operations().InsertOperation(deprovisioning)
	assert.NoError(t, err)
	rs := fixture.FixRuntimeState("rs", provisioning.RuntimeID, "prov-id")
	err = db.RuntimeStates().Insert(rs)
	assert.NoError(t, err)
	step := NewCleanStep(db.Operations(), db.RuntimeStates(), false)

	// when
	_, backoff, err := step.Run(deprovisioning, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)

	ops, err := db.Operations().ListOperationsByInstanceID("inst-id")
	assert.NoError(t, err)
	assert.Len(t, ops, 2)
	_, err = db.RuntimeStates().GetByOperationID("prov-id")
	assert.NoError(t, err)
}

func TestCleanStep_Run_ExcutedButNotCompleted(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	provisioning := fixture.FixProvisioningOperation("prov-id", "inst-id")
	deprovisioning := fixture.FixDeprovisioningOperationAsOperation("deprov-id", "inst-id")
	deprovisioning.ExcutedButNotCompleted = []string{"step1", "step2"}

	err := db.Operations().InsertOperation(provisioning)
	assert.NoError(t, err)
	err = db.Operations().InsertOperation(deprovisioning)
	assert.NoError(t, err)
	rs := fixture.FixRuntimeState("rs", provisioning.RuntimeID, "prov-id")
	err = db.RuntimeStates().Insert(rs)
	assert.NoError(t, err)
	step := NewCleanStep(db.Operations(), db.RuntimeStates(), false)

	// when
	_, backoff, err := step.Run(deprovisioning, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)

	ops, err := db.Operations().ListOperationsByInstanceID("inst-id")
	assert.NoError(t, err)
	assert.Len(t, ops, 2)
	_, err = db.RuntimeStates().GetByOperationID("prov-id")
	assert.NoError(t, err)
}
