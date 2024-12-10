package update

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInitialisationStep_OtherOperationIsInProgress(t *testing.T) {

	for tn, tc := range map[string]struct {
		beforeFunc     func(os storage.Operations)
		expectedRepeat bool
	}{
		"in progress provisioning": {
			beforeFunc: func(os storage.Operations) {
				provisioningOperation := fixture.FixProvisioningOperation("p-id", "iid")
				provisioningOperation.State = domain.InProgress
				err := os.InsertOperation(provisioningOperation)
				require.NoError(t, err)
			},
			expectedRepeat: true,
		},
		"succeeded provisioning": {
			beforeFunc: func(os storage.Operations) {
				provisioningOperation := fixture.FixProvisioningOperation("p-id", "iid")
				provisioningOperation.State = domain.Succeeded
				err := os.InsertOperation(provisioningOperation)
				require.NoError(t, err)
			},
			expectedRepeat: false,
		},
		"in progress upgrade shoot": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixUpgradeClusterOperation("op-id", "iid")
				op.State = domain.InProgress
				err := os.InsertUpgradeClusterOperation(op)
				require.NoError(t, err)
			},
			expectedRepeat: true,
		},
		"in progress update": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixUpdatingOperation("op-id", "iid")
				op.State = domain.InProgress
				err := os.InsertUpdatingOperation(op)
				require.NoError(t, err)
			},
			expectedRepeat: true,
		},
		"in progress deprovisioning": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixDeprovisioningOperation("op-id", "iid")
				op.State = domain.InProgress
				err := os.InsertDeprovisioningOperation(op)
				require.NoError(t, err)
			},
			expectedRepeat: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			db := storage.NewMemoryStorage()
			ops := db.Operations()
			is := db.Instances()
			rs := db.RuntimeStates()
			inst := fixture.FixInstance("iid")
			state := fixture.FixRuntimeState("op-id", "Runtime-iid", "op-id")
			err := is.Insert(inst)
			require.NoError(t, err)
			err = rs.Insert(state)
			require.NoError(t, err)
			builder := &automock.CreatorForPlan{}
			builder.On("CreateUpgradeShootInput", mock.Anything).
				Return(&fixture.SimpleInputCreator{}, nil)
			step := NewInitialisationStep(is, ops, builder)
			updatingOperation := fixture.FixUpdatingOperation("up-id", "iid")
			updatingOperation.State = orchestration.Pending
			err = ops.InsertOperation(updatingOperation.Operation)
			require.NoError(t, err)
			tc.beforeFunc(ops)
			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			// when
			_, d, err := step.Run(updatingOperation.Operation, log)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expectedRepeat, d != 0)
		})
	}
}
