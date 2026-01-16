package update

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
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
			inst := fixture.FixInstance("iid")
			err := is.Insert(inst)
			require.NoError(t, err)
			step := NewInitialisationStep(db)
			updatingOperation := fixture.FixUpdatingOperation("up-id", "iid")
			updatingOperation.State = internal.OperationStatePending
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
