package provisioning

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type EnableForTrialPlanStep struct {
	step process.Step
}

var _ process.Step = &EnableForTrialPlanStep{}

func NewEnableForTrialPlanStep(step process.Step) EnableForTrialPlanStep {
	return EnableForTrialPlanStep{
		step: step,
	}
}

func (s EnableForTrialPlanStep) Name() string {
	return s.step.Name()
}

func (s EnableForTrialPlanStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if !broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
		log.Info(fmt.Sprintf("Skipping step %s", s.Name()))
		return operation, 0, nil
	}

	return s.step.Run(operation, log)
}
