package steps

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"log/slog"
	"time"
)

type HolderStep struct {
	holdMode bool
	target   process.Step
}

func NewHolderStep(holdMode bool, target process.Step) *HolderStep {
	return &HolderStep{
		holdMode: holdMode,
		target:   target,
	}
}

func (h HolderStep) Name() string {
	return h.target.Name()
}

func (h HolderStep) Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error) {
	if h.holdMode {
		logger.Info("Retrying due to blocking HAP mode")
		return operation, 1 * time.Minute, nil
	}
	return h.target.Run(operation, logger)
}

var _ process.Step = &HolderStep{}
