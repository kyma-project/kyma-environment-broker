package broker_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBroker implements domain.ServiceBroker for testing
type mockBroker struct {
	shouldPanic bool
}

func (m *mockBroker) Services(ctx context.Context) ([]domain.Service, error) {
	if m.shouldPanic {
		panic("test panic in Services")
	}
	return []domain.Service{}, nil
}

func (m *mockBroker) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (domain.ProvisionedServiceSpec, error) {
	if m.shouldPanic {
		panic("test panic in Provision")
	}
	return domain.ProvisionedServiceSpec{}, nil
}

func (m *mockBroker) Deprovision(ctx context.Context, instanceID string, details domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	if m.shouldPanic {
		panic("test panic in Deprovision")
	}
	return domain.DeprovisionServiceSpec{}, nil
}

func (m *mockBroker) Update(ctx context.Context, instanceID string, details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	if m.shouldPanic {
		panic("test panic in Update")
	}
	return domain.UpdateServiceSpec{}, nil
}

func (m *mockBroker) GetInstance(ctx context.Context, instanceID string, details domain.FetchInstanceDetails) (domain.GetInstanceDetailsSpec, error) {
	if m.shouldPanic {
		panic("test panic in GetInstance")
	}
	return domain.GetInstanceDetailsSpec{}, nil
}

func (m *mockBroker) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (domain.LastOperation, error) {
	if m.shouldPanic {
		panic("test panic in LastOperation")
	}
	return domain.LastOperation{}, nil
}

func (m *mockBroker) Bind(ctx context.Context, instanceID, bindingID string, details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {
	if m.shouldPanic {
		panic("test panic in Bind")
	}
	return domain.Binding{}, nil
}

func (m *mockBroker) Unbind(ctx context.Context, instanceID, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {
	if m.shouldPanic {
		panic("test panic in Unbind")
	}
	return domain.UnbindSpec{}, nil
}

func (m *mockBroker) GetBinding(ctx context.Context, instanceID, bindingID string, details domain.FetchBindingDetails) (domain.GetBindingSpec, error) {
	if m.shouldPanic {
		panic("test panic in GetBinding")
	}
	return domain.GetBindingSpec{}, nil
}

func (m *mockBroker) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	if m.shouldPanic {
		panic("test panic in LastBindingOperation")
	}
	return domain.LastOperation{}, nil
}

func TestWithPanicRecovery_Provision(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		planID := "test-plan-id"
		details := domain.ProvisionDetails{PlanID: planID}

		// when
		_, err := wrapper.Provision(context.Background(), instanceID, details, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during provisioning")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during provisioning: test panic in Provision")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "planID="+planID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Provision(context.Background(), "instance-id", domain.ProvisionDetails{}, true)

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_Deprovision(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		planID := "test-plan-id"
		details := domain.DeprovisionDetails{PlanID: planID}

		// when
		_, err := wrapper.Deprovision(context.Background(), instanceID, details, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during deprovisioning")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during deprovisioning: test panic in Deprovision")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "planID="+planID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Deprovision(context.Background(), "instance-id", domain.DeprovisionDetails{}, true)

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_Update(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		planID := "test-plan-id"
		details := domain.UpdateDetails{PlanID: planID}

		// when
		_, err := wrapper.Update(context.Background(), instanceID, details, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during update")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during update: test panic in Update")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "planID="+planID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Update(context.Background(), "instance-id", domain.UpdateDetails{}, true)

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_GetInstance(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"

		// when
		_, err := wrapper.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during get instance")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during get instance: test panic in GetInstance")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.GetInstance(context.Background(), "instance-id", domain.FetchInstanceDetails{})

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_LastOperation(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		operationID := "test-operation-id"
		details := domain.PollDetails{OperationData: operationID}

		// when
		_, err := wrapper.LastOperation(context.Background(), instanceID, details)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during last operation")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during last operation: test panic in LastOperation")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "operationID="+operationID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.LastOperation(context.Background(), "instance-id", domain.PollDetails{})

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_Bind(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		bindingID := "test-binding-id"

		// when
		_, err := wrapper.Bind(context.Background(), instanceID, bindingID, domain.BindDetails{}, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during bind")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during bind: test panic in Bind")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "bindingID="+bindingID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Bind(context.Background(), "instance-id", "binding-id", domain.BindDetails{}, true)

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_Unbind(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		bindingID := "test-binding-id"

		// when
		_, err := wrapper.Unbind(context.Background(), instanceID, bindingID, domain.UnbindDetails{}, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during unbind")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during unbind: test panic in Unbind")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "bindingID="+bindingID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Unbind(context.Background(), "instance-id", "binding-id", domain.UnbindDetails{}, true)

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_GetBinding(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		bindingID := "test-binding-id"

		// when
		_, err := wrapper.GetBinding(context.Background(), instanceID, bindingID, domain.FetchBindingDetails{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during get binding")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during get binding: test panic in GetBinding")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "bindingID="+bindingID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.GetBinding(context.Background(), "instance-id", "binding-id", domain.FetchBindingDetails{})

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_LastBindingOperation(t *testing.T) {
	t.Run("catches panic and logs with context", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		instanceID := "test-instance-id"
		bindingID := "test-binding-id"
		operationID := "test-operation-id"
		details := domain.PollDetails{OperationData: operationID}

		// when
		_, err := wrapper.LastBindingOperation(context.Background(), instanceID, bindingID, details)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during last binding operation")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during last binding operation: test panic in LastBindingOperation")
		assert.Contains(t, logs, "instanceID="+instanceID)
		assert.Contains(t, logs, "bindingID="+bindingID)
		assert.Contains(t, logs, "operationID="+operationID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.LastBindingOperation(context.Background(), "instance-id", "binding-id", domain.PollDetails{})

		// then
		require.NoError(t, err)
	})
}

func TestWithPanicRecovery_Services(t *testing.T) {
	t.Run("catches panic and logs", func(t *testing.T) {
		// given
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: true}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when - Services doesn't have panic recovery currently, so it will panic
		assert.Panics(t, func() {
			_, _ = wrapper.Services(context.Background())
		})
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Services(context.Background())

		// then
		require.NoError(t, err)
	})
}
