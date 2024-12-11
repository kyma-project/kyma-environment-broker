package upgrade_cluster

import (
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type LogSkippingUpgradeStep struct {
	operationManager *process.UpgradeClusterOperationManager
}

func (s *LogSkippingUpgradeStep) Name() string {
	return "Log_Skipping_Upgrade"
}

func NewLogSkippingUpgradeStep(os storage.Operations) *LogSkippingUpgradeStep {
	return &LogSkippingUpgradeStep{
		operationManager: process.NewUpgradeClusterOperationManager(os),
	}
}

func (s *LogSkippingUpgradeStep) Run(operation internal.UpgradeClusterOperation, log *slog.Logger) (internal.UpgradeClusterOperation, time.Duration, error) {
	log.Info("Skipping cluster upgrade due to step condition not met")

	return s.operationManager.OperationSucceeded(operation, "upgrade cluster skipped due to step condition", log)
}
