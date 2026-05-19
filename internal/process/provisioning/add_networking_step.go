package provisioning

import (
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type AddNetworkingStep struct {
	operationManager *process.OperationManager
}

func NewAddNetworkingStep(os storage.Operations) *AddNetworkingStep {
	step := &AddNetworkingStep{}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.KEBDependency)
	return step
}

func (s *AddNetworkingStep) Name() string {
	return "Add_Networking"
}

func (s *AddNetworkingStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	// TODO: implement
	return operation, 0, nil
}
