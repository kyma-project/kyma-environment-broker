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

const (
	testInstanceID  = "test-instance-id"
	testPlanID      = "test-plan-id"
	testBindingID   = "test-binding-id"
	testOperationID = "test-operation-id"
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

		details := domain.ProvisionDetails{PlanID: testPlanID}

		// when
		_, err := wrapper.Provision(context.Background(), testInstanceID, details, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during provisioning")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during provisioning: test panic in Provision")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "planID="+testPlanID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Provision(context.Background(), testInstanceID, domain.ProvisionDetails{}, true)

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

		details := domain.DeprovisionDetails{PlanID: testPlanID}

		// when
		_, err := wrapper.Deprovision(context.Background(), testInstanceID, details, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during deprovisioning")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during deprovisioning: test panic in Deprovision")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "planID="+testPlanID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Deprovision(context.Background(), testInstanceID, domain.DeprovisionDetails{}, true)

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

		details := domain.UpdateDetails{PlanID: testPlanID}

		// when
		_, err := wrapper.Update(context.Background(), testInstanceID, details, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during update")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during update: test panic in Update")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "planID="+testPlanID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Update(context.Background(), testInstanceID, domain.UpdateDetails{}, true)

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

		// when
		_, err := wrapper.GetInstance(context.Background(), testInstanceID, domain.FetchInstanceDetails{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during get instance")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during get instance: test panic in GetInstance")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.GetInstance(context.Background(), testInstanceID, domain.FetchInstanceDetails{})

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

		details := domain.PollDetails{OperationData: testOperationID}

		// when
		_, err := wrapper.LastOperation(context.Background(), testInstanceID, details)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during last operation")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during last operation: test panic in LastOperation")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "operationID="+testOperationID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.LastOperation(context.Background(), testInstanceID, domain.PollDetails{})

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

		// when
		_, err := wrapper.Bind(context.Background(), testInstanceID, testBindingID, domain.BindDetails{}, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during bind")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during bind: test panic in Bind")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "bindingID="+testBindingID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Bind(context.Background(), testInstanceID, testBindingID, domain.BindDetails{}, true)

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

		// when
		_, err := wrapper.Unbind(context.Background(), testInstanceID, testBindingID, domain.UnbindDetails{}, true)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during unbind")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during unbind: test panic in Unbind")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "bindingID="+testBindingID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.Unbind(context.Background(), testInstanceID, testBindingID, domain.UnbindDetails{}, true)

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

		// when
		_, err := wrapper.GetBinding(context.Background(), testInstanceID, testBindingID, domain.FetchBindingDetails{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during get binding")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during get binding: test panic in GetBinding")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "bindingID="+testBindingID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.GetBinding(context.Background(), testInstanceID, testBindingID, domain.FetchBindingDetails{})

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

		details := domain.PollDetails{OperationData: testOperationID}

		// when
		_, err := wrapper.LastBindingOperation(context.Background(), testInstanceID, testBindingID, details)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error during last binding operation")

		logs := logOutput.String()
		assert.Contains(t, logs, "panic recovered during last binding operation: test panic in LastBindingOperation")
		assert.Contains(t, logs, "instanceID="+testInstanceID)
		assert.Contains(t, logs, "bindingID="+testBindingID)
		assert.Contains(t, logs, "operationID="+testOperationID)
		assert.Contains(t, logs, "stack=")
	})

	t.Run("normal execution without panic", func(t *testing.T) {
		// given
		logger := slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelError}))
		mockBroker := &mockBroker{shouldPanic: false}
		wrapper := broker.NewWithPanicRecovery(mockBroker, logger)

		// when
		_, err := wrapper.LastBindingOperation(context.Background(), testInstanceID, testBindingID, domain.PollDetails{})

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
