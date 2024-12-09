package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
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

			kimConfig := broker.KimConfig{
				Enabled: false,
			}

			st := storage.NewMemoryStorage()
			operation := fixOperationRuntimeStatus(broker.GCPPlanID, pkg.GCP)
			operation.RuntimeID = statusRuntimeID
			operation.DashboardURL = dashboardURL
			err := st.Operations().InsertOperation(operation)
			assert.NoError(t, err)

			step := NewCheckRuntimeStep(st.Operations(), provisionerClient, time.Second, kimConfig)

			// when
			operation, repeat, err := step.Run(operation, fixLogger())

			// then
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedRepeat, repeat > 0)
			assert.Equal(t, domain.InProgress, operation.State)
		})
	}
}

func TestCheckRuntimeStep_RunProvisioningSucceeded_WithKimOnly(t *testing.T) {
	for _, tc := range []struct {
		name              string
		provisionerStatus gqlschema.OperationState
		expectedRepeat    bool
	}{
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

			kimConfig := broker.KimConfig{
				Enabled:      true,
				Plans:        []string{"gcp"},
				KimOnlyPlans: []string{"gcp"},
			}

			st := storage.NewMemoryStorage()
			operation := fixOperationRuntimeStatus(broker.GCPPlanID, pkg.GCP)
			operation.RuntimeID = statusRuntimeID
			operation.DashboardURL = dashboardURL
			err := st.Operations().InsertOperation(operation)
			assert.NoError(t, err)

			step := NewCheckRuntimeStep(st.Operations(), provisionerClient, time.Second, kimConfig)

			// when
			operation, repeat, err := step.Run(operation, fixLogger())

			// then
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedRepeat, repeat > 0)
			assert.Equal(t, domain.InProgress, operation.State)
		})
	}
}
