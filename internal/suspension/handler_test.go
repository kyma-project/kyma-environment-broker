package suspension

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	shootNameC012345   = "c-012345"
	shootDomainC012345 = "c-012345.sap.com"
)

func TestSuspension(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, fixLogger())
	instance := fixInstance(fixActiveErsContext())
	err := st.Instances().Insert(*instance)
	require.NoError(t, err)

	// when
	changed, err := svc.Handle(instance, fixInactiveErsContext())
	require.NoError(t, err)
	assert.True(t, changed, "handler to change active flag")

	// then
	op, _ := st.Operations().GetDeprovisioningOperationByInstanceID("instance-id")
	assertQueue(t, deprovisioning, op.ID)
	assertQueue(t, provisioning)

	assert.Equal(t, domain.LastOperationState("pending"), op.State)
	assert.Equal(t, instance.InstanceID, op.InstanceID)
}

func TestSuspension_Retrigger(t *testing.T) {
	t.Run("should skip suspension when temporary deprovisioning operation already succeeded", func(t *testing.T) {
		// given
		provisioning := NewDummyQueue()
		deprovisioning := NewDummyQueue()
		st := storage.NewMemoryStorage()

		svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, fixLogger())
		instance := fixInstance(fixInactiveErsContext())
		err := st.Instances().Insert(*instance)
		require.NoError(t, err)
		err = st.Operations().InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         "suspended-op-id",
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: instance.InstanceID,
				State:      domain.Succeeded,
				Temporary:  true,
				Type:       internal.OperationTypeDeprovision,
			},
		})
		require.NoError(t, err)

		// when
		changed, err := svc.Handle(instance, fixInactiveErsContext())
		require.NoError(t, err)
		assert.False(t, changed, "handler to not change active flag")

		// then
		op, _ := st.Operations().GetDeprovisioningOperationByInstanceID("instance-id")
		assertQueue(t, deprovisioning)
		assertQueue(t, provisioning)

		assert.Equal(t, domain.Succeeded, op.State)
		assert.Equal(t, instance.InstanceID, op.InstanceID)
	})

	t.Run("should retrigger deprovisioning when existing temporary deprovisioning operation failed", func(t *testing.T) {
		// given
		provisioning := NewDummyQueue()
		deprovisioning := NewDummyQueue()
		st := storage.NewMemoryStorage()

		svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, fixLogger())
		instance := fixInstance(fixInactiveErsContext())
		err := st.Instances().Insert(*instance)
		require.NoError(t, err)
		err = st.Operations().InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         "suspended-op-id",
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: instance.InstanceID,
				State:      domain.Failed,
				Temporary:  true,
				Type:       internal.OperationTypeDeprovision,
			},
		})
		require.NoError(t, err)

		// when
		changed, err := svc.Handle(instance, fixInactiveErsContext())
		require.NoError(t, err)
		assert.True(t, changed, "handler to change active flag")

		// then
		op, _ := st.Operations().GetDeprovisioningOperationByInstanceID("instance-id")
		assertQueue(t, deprovisioning, op.ID)
		assertQueue(t, provisioning)

		assert.Equal(t, domain.LastOperationState("pending"), op.State)
		assert.Equal(t, instance.InstanceID, op.InstanceID)
	})

}

func assertQueue(t *testing.T, q *dummyQueue, id ...string) {
	t.Helper()
	if len(id) == 0 {
		assert.Empty(t, q.IDs)
		return
	}
	assert.Equal(t, q.IDs, id)
}

func TestUnsuspension(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, fixLogger())
	instance := fixInstance(fixInactiveErsContext())
	instance.InstanceDetails.ShootName = shootNameC012345
	instance.InstanceDetails.ShootDomain = shootDomainC012345

	err := st.Instances().Insert(*instance)
	require.NoError(t, err)

	deprovisioningOperation := fixture.FixDeprovisioningOperation("d-op", "instance-id")
	deprovisioningOperation.Temporary = true
	err = st.Operations().InsertDeprovisioningOperation(deprovisioningOperation)
	require.NoError(t, err)

	// when
	changed, err := svc.Handle(instance, fixActiveErsContext())
	require.NoError(t, err)
	assert.True(t, changed, "handler to change active flag")

	// then
	op, err := st.Operations().GetProvisioningOperationByInstanceID("instance-id")
	require.NoError(t, err)
	assertQueue(t, deprovisioning)
	assertQueue(t, provisioning, op.ID)

	assert.Equal(t, domain.LastOperationState(internal.OperationStatePending), op.State)
	assert.Equal(t, instance.InstanceID, op.InstanceID)
	assert.Equal(t, shootNameC012345, op.ShootName)
	assert.Equal(t, shootDomainC012345, op.ShootDomain)
}

func TestUnsuspensionForDeprovisioningInstance(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, fixLogger())
	instance := fixInstance(fixInactiveErsContext())
	instance.InstanceDetails.ShootName = shootNameC012345
	instance.InstanceDetails.ShootDomain = shootDomainC012345

	err := st.Instances().Insert(*instance)
	require.NoError(t, err)
	deprovisioningOperation := fixture.FixDeprovisioningOperation("d-op", "instance-id")
	deprovisioningOperation.Temporary = false
	err = st.Operations().InsertDeprovisioningOperation(deprovisioningOperation)
	require.NoError(t, err)

	// when
	changed, err := svc.Handle(instance, fixActiveErsContext())
	require.NoError(t, err)
	assert.False(t, changed, "handler to not change active flag")

	// then
	_, err = st.Operations().GetProvisioningOperationByInstanceID("instance-id")
	assert.True(t, dberr.IsNotFound(err))
	assertQueue(t, deprovisioning)
	assertQueue(t, provisioning)
}

func TestUnsuspensionForExpiredInstance(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, fixLogger())
	instance := fixInstance(fixInactiveErsContext())
	instance.InstanceDetails.ShootName = shootNameC012345
	instance.InstanceDetails.ShootDomain = shootDomainC012345
	instance.ExpiredAt = ptr.Time(time.Now())

	err := st.Instances().Insert(*instance)
	require.NoError(t, err)

	// when
	changed, err := svc.Handle(instance, fixActiveErsContext())
	require.NoError(t, err)
	assert.False(t, changed, "handler to not change active flag")

	// then
	_, err = st.Operations().GetProvisioningOperationByInstanceID("instance-id")
	assert.True(t, dberr.IsNotFound(err))
}

func fixInstance(ersContext internal.ERSContext) *internal.Instance {
	instance := fixture.FixInstance("instance-id")
	instance.ServicePlanID = broker.TrialPlanID
	instance.Parameters.ErsContext = ersContext

	return &instance
}

func fixActiveErsContext() internal.ERSContext {
	return internal.ERSContext{
		Active: ptr.Bool(true),
	}
}

func fixInactiveErsContext() internal.ERSContext {
	return internal.ERSContext{
		Active: ptr.Bool(false),
	}
}

type dummyQueue struct {
	IDs []string
}

func NewDummyQueue() *dummyQueue {
	return &dummyQueue{
		IDs: []string{},
	}
}

func (q *dummyQueue) Add(id string) {
	q.IDs = append(q.IDs, id)
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
