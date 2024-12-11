package upgrade_cluster

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/kyma-environment-broker/internal/notification/mocks"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestSendNotificationStep_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	tenants := []notification.NotificationTenant{
		{
			InstanceID: notification.FakeInstanceID,
			StartDate:  time.Now().Format("2006-01-02 15:04:05"),
			State:      notification.UnderMaintenanceEventState,
		},
	}
	paras := notification.NotificationParams{
		OrchestrationID: notification.FakeOrchestrationID,
		Tenants:         tenants,
	}

	bundleBuilder := &mocks.BundleBuilder{}
	bundle := &mocks.Bundle{}
	bundleBuilder.On("NewBundle", notification.FakeOrchestrationID, paras).Return(bundle, nil).Once()
	bundle.On("UpdateNotificationEvent").Return(nil).Once()

	operation := internal.UpgradeClusterOperation{
		Operation: internal.Operation{
			InstanceID:      notification.FakeInstanceID,
			OrchestrationID: notification.FakeOrchestrationID,
		},
	}
	step := NewSendNotificationStep(memoryStorage.Operations(), bundleBuilder)

	// when
	_, repeat, err := step.Run(operation, log)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
}
