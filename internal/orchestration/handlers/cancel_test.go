package handlers

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixOrchestrationID = "test-id"
)

func TestCanceler_CancelForID(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	t.Run("should cancel orchestration", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		err := s.Orchestrations().Insert(fixOrchestration())
		require.NoError(t, err)

		c := NewCanceler(s.Orchestrations(), log)

		err = c.CancelForID(fixOrchestrationID)
		require.NoError(t, err)

		isCanceling, err := isCanceling(s.Orchestrations())
		require.NoError(t, err)

		assert.True(t, isCanceling)
	})
	t.Run("already canceling", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		o := fixOrchestration()
		o.State = orchestration.Canceling
		err := s.Orchestrations().Insert(o)
		require.NoError(t, err)

		c := NewCanceler(s.Orchestrations(), log)

		err = c.CancelForID(fixOrchestrationID)
		require.NoError(t, err)

		isCanceling, err := isCanceling(s.Orchestrations())
		require.NoError(t, err)

		assert.True(t, isCanceling)
	})
	t.Run("already canceled", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		o := fixOrchestration()
		o.State = orchestration.Canceled
		err := s.Orchestrations().Insert(o)
		require.NoError(t, err)

		c := NewCanceler(s.Orchestrations(), log)

		err = c.CancelForID(fixOrchestrationID)
		require.NoError(t, err)

		isCanceling, err := isCanceling(s.Orchestrations())
		require.NoError(t, err)

		assert.False(t, isCanceling)
	})
	t.Run("should return error when orchestration not found", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		c := NewCanceler(s.Orchestrations(), log)

		err := c.CancelForID(fixOrchestrationID)
		assert.Error(t, err)
	})
}

func isCanceling(s storage.Orchestrations) (bool, error) {
	o, err := s.GetByID(fixOrchestrationID)
	if err != nil {
		return false, err
	}
	if o.State == orchestration.Canceling {
		return true, nil
	}
	return false, nil
}

func fixOrchestration() internal.Orchestration {
	o := fixture.FixOrchestration(fixOrchestrationID)
	o.State = orchestration.InProgress

	return o
}
