package steps_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/stretchr/testify/assert"
)

func TestHolderStep_Run(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	t.Run("hold mode enabled", func(t *testing.T) {
		// Given
		mockStep := new(MockStep)
		holderStep := steps.NewHolderStep(true, mockStep)
		operation := internal.Operation{}

		// When
		resultOperation, repeat, err := holderStep.Run(operation, logger)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, operation, resultOperation)
		assert.Equal(t, 1*time.Minute, repeat)
		mockStep.assertRunNotCalled(t)
	})

	t.Run("hold mode disabled", func(t *testing.T) {
		// Given
		mockStep := new(MockStep)
		holderStep := steps.NewHolderStep(false, mockStep)
		operation := internal.Operation{}

		// When
		_, repeat, err := holderStep.Run(operation, logger)

		// Then
		assert.NoError(t, err)
		assert.Zero(t, repeat)
		mockStep.assertRunCalled(t)
	})
}

// MockStep is a mock implementation of the process.Step interface
type MockStep struct {
	called bool
}

func (m *MockStep) Name() string {
	return "mock-step"
}

func (m *MockStep) Run(operation internal.Operation, _ *slog.Logger) (internal.Operation, time.Duration, error) {
	m.called = true
	return operation, 0, nil
}

func (m *MockStep) assertRunCalled(t *testing.T) {
	assert.True(t, m.called, "Expected Run to be called, but it was not")
}

func (m *MockStep) assertRunNotCalled(t *testing.T) {
	assert.False(t, m.called, "Expected Run not to be called, but it was")
}
