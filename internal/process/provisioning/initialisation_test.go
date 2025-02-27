package provisioning

import (
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	automock2 "github.com/kyma-project/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
)

const (
	statusOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	statusInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	statusRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	statusGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	statusProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"

	dashboardURL = "http://runtime.com"
)

func TestInitialisationStep_Run(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	operation := fixOperationRuntimeStatus(broker.GCPPlanID, pkg.GCP)
	err := st.Operations().InsertOperation(operation)
	assert.NoError(t, err)
	err = st.Instances().Insert(fixture.FixInstance(operation.InstanceID))
	assert.NoError(t, err)
	ri := &simpleInputCreator{
		provider: pkg.GCP,
		config: &internal.ConfigForPlan{
			KymaTemplate: "kyma-template",
		},
	}
	builder := &automock2.CreatorForPlan{}
	builder.On("CreateProvisionInput", operation.ProvisioningParameters).Return(ri, nil)

	step := NewInitialisationStep(st.Operations(), st.Instances(), builder)

	// when
	op, retry, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	assert.Equal(t, ri, op.InputCreator)

	inst, _ := st.Instances().GetByID(operation.InstanceID)
	// make sure the provider is saved into the instance
	assert.Equal(t, pkg.GCP, inst.Provider)
}

func fixOperationRuntimeStatus(planId string, provider pkg.CloudProvider) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	provisioningOperation.State = domain.InProgress
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.InstanceDetails.RuntimeID = runtimeID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID

	return provisioningOperation
}

func fixOperationRuntimeStatusWithProvider(planId string, provider pkg.CloudProvider) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	provisioningOperation.State = ""
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID
	provisioningOperation.ProvisioningParameters.Parameters.Provider = &provider

	return provisioningOperation
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(statusInstanceID)
	instance.RuntimeID = statusRuntimeID
	instance.DashboardURL = dashboardURL
	instance.GlobalAccountID = statusGlobalAccountID

	return instance
}
