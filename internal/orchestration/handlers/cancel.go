package handlers

import (
	"fmt"
	"log/slog"
	"time"

	orchestrationExt "github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type Canceler struct {
	orchestrations storage.Orchestrations
	log            *slog.Logger
}

func NewCanceler(orchestrations storage.Orchestrations, logger *slog.Logger) *Canceler {
	return &Canceler{
		orchestrations: orchestrations,
		log:            logger,
	}
}

// CancelForID cancels orchestration by ID
func (c *Canceler) CancelForID(orchestrationID string) error {
	o, err := c.orchestrations.GetByID(orchestrationID)
	if err != nil {
		return fmt.Errorf("while getting orchestration: %w", err)
	}
	if o.IsFinished() || o.State == orchestrationExt.Canceling {
		return nil
	}

	o.UpdatedAt = time.Now()
	o.Description = "Orchestration was canceled"
	o.State = orchestrationExt.Canceling
	err = c.orchestrations.Update(*o)
	if err != nil {
		return fmt.Errorf("while updating orchestration: %w", err)
	}
	return nil
}
