package update

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

const (
	statusProvisionerOperationID = "194ae524-5343-489b-b05a-296be593e6cf"
	statusRuntimeID              = "runtime-id"
)

func TestCheckRuntimeStep_RunProvisioningSucceeded(t *testing.T) {
	for _, tc := range []struct {
		name              string
		provisionerStatus gqlschema.OperationState
		expectedRepeat    bool
	}{
		{
			name:              "In Progress",
			provisionerStatus: gqlschema.OperationStateInProgress,
			expectedRepeat:    true,
		},
		{
			name:              "Succeeded",
			provisionerStatus: gqlschema.OperationStateSucceeded,
			expectedRepeat:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given
			provisionerClient := provisioner.NewFakeClient()
			provisionerClient.SetOperation(statusProvisionerOperationID, gqlschema.OperationStatus{
				ID:        ptr.String(statusProvisionerOperationID),
				Operation: gqlschema.OperationTypeProvision,
				State:     tc.provisionerStatus,
				Message:   nil,
				RuntimeID: ptr.String(statusRuntimeID),
			})
			st := storage.NewMemoryStorage()
			operation := fixOperationRuntimeStatus(broker.GCPPlanID)
			operation.RuntimeID = statusRuntimeID
			err := st.Operations().InsertOperation(operation)
			assert.NoError(t, err)
			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			step := NewCheckStep(st.Operations(), provisionerClient, 1*time.Second)

			// when
			operation, repeat, err := step.Run(operation, log)

			// then
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedRepeat, repeat > 0)
			assert.Equal(t, domain.InProgress, operation.State)
		})
	}
}

func fixOperationRuntimeStatus(id string) internal.Operation {
	return internal.Operation{
		ID:                     id,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		ProvisionerOperationID: statusProvisionerOperationID,
		State:                  domain.InProgress,
		UpdatingParameters:     internal.UpdatingParametersDTO{},
		InputCreator:           nil,
	}
}
