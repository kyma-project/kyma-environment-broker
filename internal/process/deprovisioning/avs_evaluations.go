package deprovisioning

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/process"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type AvsEvaluationRemovalStep struct {
	delegator             *avs.Delegator
	externalEvalAssistant avs.EvalAssistant
	internalEvalAssistant avs.EvalAssistant
	deProvisioningManager *process.OperationManager
}

func NewAvsEvaluationsRemovalStep(delegator *avs.Delegator, operationsStorage storage.Operations, externalEvalAssistant, internalEvalAssistant avs.EvalAssistant) *AvsEvaluationRemovalStep {
	return &AvsEvaluationRemovalStep{
		delegator:             delegator,
		externalEvalAssistant: externalEvalAssistant,
		internalEvalAssistant: internalEvalAssistant,
		deProvisioningManager: process.NewOperationManager(operationsStorage),
	}
}

func (ars *AvsEvaluationRemovalStep) Name() string {
	return "De-provision_AVS_Evaluations"
}

func (ars *AvsEvaluationRemovalStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	logger.Infof("Avs lifecycle %+v", operation.Avs)

	// the delegator saves the operation (update in the storage) - it is executed inside the DeleteAvsEvaluation

	operation, err := ars.delegator.DeleteAvsEvaluation(operation, logger, ars.internalEvalAssistant)
	if err != nil {
		logger.Warnf("unable to delete internal evaluation: %s", err.Error())
		return ars.deProvisioningManager.RetryOperationWithoutFail(operation, ars.Name(), "error while deleting avs internal evaluation", 10*time.Second, 1*time.Minute, logger)
	}

	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) || broker.IsFreemiumPlan(operation.ProvisioningParameters.PlanID) {
		logger.Info("skipping AVS external evaluation deletion for trial/freemium plan")
		return operation, 0, nil
	}
	operation, err = ars.delegator.DeleteAvsEvaluation(operation, logger, ars.externalEvalAssistant)
	if err != nil {
		logger.Warnf("unable to delete external evaluation: %s", err.Error())
		return ars.deProvisioningManager.RetryOperationWithoutFail(operation, ars.Name(), "error while deleting avs external evaluation", 10*time.Second, 1*time.Minute, logger)
	}

	return operation, 0, nil

}
