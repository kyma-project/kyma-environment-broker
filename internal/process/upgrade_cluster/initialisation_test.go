package upgrade_cluster

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process/upgrade_cluster/automock"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	notificationAutomock "github.com/kyma-project/kyma-environment-broker/internal/notification/mocks"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	cloudProvider "github.com/kyma-project/kyma-environment-broker/internal/provider"
	provisionerAutomock "github.com/kyma-project/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	fixProvisioningOperationID                      = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	fixOrchestrationID                              = "fd5cee4d-0eeb-40d0-a7a7-0708eseba470"
	fixUpgradeOperationID                           = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	fixInstanceID                                   = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	fixRuntimeID                                    = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	fixGlobalAccountID                              = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	fixSubAccountID                                 = "6424cc6d-5fce-49fc-b720-cf1fc1f36c7d"
	fixProvisionerOperationID                       = "e04de524-53b3-4890-b05a-296be393e4ba"
	fixMaintenanceModeAlwaysDisabledGlobalAccountID = "maintenance-mode-always-disabled-ga-1"
)

type fixHyperscalerInputProvider interface {
	Defaults() *gqlschema.ClusterConfigInput
}

func TestInitialisationStep_Run(t *testing.T) {
	t.Run("should mark operation as Succeeded when upgrade was successful", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		orch := internal.Orchestration{
			OrchestrationID: fixOrchestrationID,
			State:           orchestration.InProgress,
			Parameters: orchestration.Parameters{
				Notification: true,
			},
		}
		err := memoryStorage.Orchestrations().Insert(orch)
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()
		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient,
			nil, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)

	})

	t.Run("should initialize UpgradeRuntimeInput request when run", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()
		upgradeOperation.ProvisionerOperationID = ""
		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		inputBuilder := &automock.CreatorForPlan{}
		inputBuilder.On("CreateUpgradeShootInput", fixProvisioningParameters()).
			Return(&input.RuntimeInput{},
				nil)

		expectedOperation := upgradeOperation
		expectedOperation.Version++
		expectedOperation.State = orchestration.InProgress

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		op, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		inputBuilder.AssertNumberOfCalls(t, "CreateUpgradeShootInput", 1)
		assert.Equal(t, time.Duration(0), repeat)
		assert.NotNil(t, op.InputCreator)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(op.Operation.ID)
		op.InputCreator = nil
		assert.Equal(t, op, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should mark finish if orchestration was canceled", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{
			OrchestrationID: fixOrchestrationID,
			State:           orchestration.Canceled,
			Parameters: orchestration.Parameters{
				Notification: true,
			},
		})
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()
		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), nil, nil, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, orchestration.Canceled, string(upgradeOperation.State))

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		require.NoError(t, err)
		assert.Equal(t, upgradeOperation, *storedOp)
	})

	t.Run("should refresh avs on success (both monitors, empty init)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (both monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on fail (both monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateFailed,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NotNil(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Failed, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (internal monitor)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (external monitor)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (no monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(fixOrchestrationWithKymaVer())
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		notificationTenants := []notification.NotificationTenant{
			{
				InstanceID: fixInstanceID,
				State:      notification.FinishedMaintenanceState,
				EndDate:    time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		notificationParas := notification.NotificationParams{
			OrchestrationID: fixOrchestrationID,
			Tenants:         notificationTenants,
		}
		notificationBuilder := &notificationAutomock.BundleBuilder{}
		bundle := &notificationAutomock.Bundle{}
		notificationBuilder.On("NewBundle", fixOrchestrationID, notificationParas).Return(bundle, nil).Once()
		bundle.On("UpdateNotificationEvent").Return(nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, nil, notificationBuilder)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})
}

func fixUpgradeClusterOperation() internal.UpgradeClusterOperation {
	return fixUpgradeClusterOperationWithAvs(internal.AvsLifecycleData{})
}

func fixUpgradeClusterOperationWithAvs(avsData internal.AvsLifecycleData) internal.UpgradeClusterOperation {
	upgradeOperation := fixture.FixUpgradeClusterOperation(fixUpgradeOperationID, fixInstanceID)
	upgradeOperation.OrchestrationID = fixOrchestrationID
	upgradeOperation.ProvisionerOperationID = fixProvisionerOperationID
	upgradeOperation.State = orchestration.Pending
	upgradeOperation.Description = ""
	upgradeOperation.UpdatedAt = time.Now()
	upgradeOperation.InstanceDetails.Avs = avsData
	upgradeOperation.ProvisioningParameters = fixProvisioningParameters()
	upgradeOperation.RuntimeOperation.GlobalAccountID = fixGlobalAccountID
	upgradeOperation.RuntimeOperation.SubAccountID = fixSubAccountID
	upgradeOperation.InputCreator = nil

	return upgradeOperation
}

func fixProvisioningOperation() internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperation(fixProvisioningOperationID, fixInstanceID)
	provisioningOperation.ProvisionerOperationID = fixProvisionerOperationID
	provisioningOperation.Description = ""
	provisioningOperation.ProvisioningParameters = fixProvisioningParameters()

	return provisioningOperation
}

func fixProvisioningParameters() internal.ProvisioningParameters {
	pp := fixture.FixProvisioningParameters("1")
	pp.PlanID = broker.AzurePlanID
	pp.ServiceID = ""
	pp.ErsContext.GlobalAccountID = fixGlobalAccountID
	pp.ErsContext.SubAccountID = fixSubAccountID

	return pp
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(fixInstanceID)
	instance.RuntimeID = fixRuntimeID
	instance.GlobalAccountID = fixGlobalAccountID

	return instance
}

func StringPtr(s string) *string {
	return &s
}

// no forFreemiumPlan and forTrialPlan supported for the mock testing. planID by default is broker.AzurePlanID.
func fixGetHyperscalerProviderForPlanID(planID string) fixHyperscalerInputProvider {
	var provider fixHyperscalerInputProvider
	switch planID {
	case broker.GCPPlanID:
		provider = &cloudProvider.GcpInput{}
	case broker.SapConvergedCloudPlanID:
		provider = &cloudProvider.SapConvergedCloudInput{}
	case broker.AzurePlanID:
		provider = &cloudProvider.AzureInput{}
	case broker.AzureLitePlanID:
		provider = &cloudProvider.AzureLiteInput{}
	case broker.AWSPlanID:
		provider = &cloudProvider.AWSInput{}
		// insert cases for other providers like AWS or GCP
	default:
		return nil
	}
	return provider
}

func fixOrchestrationWithKymaVer() internal.Orchestration {
	orch := internal.Orchestration{
		OrchestrationID: fixOrchestrationID,
		State:           orchestration.InProgress,
		Parameters: orchestration.Parameters{
			Notification: true,
		},
	}
	return orch
}
