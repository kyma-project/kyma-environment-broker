package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/sirupsen/logrus"
)

const (
	// label key used to send to director
	grafanaURLLabel = "operator_grafanaUrl"
)

//go:generate mockery --name=DirectorClient --output=automock --outpkg=automock --case=underscore

type DirectorClient interface {
	SetLabel(accountID, runtimeID, key, value string) error
}

type KymaVersionConfigurator interface {
	ForGlobalAccount(string) (string, bool, error)
}

type InitialisationStep struct {
	operationManager *process.OperationManager
	inputBuilder     input.CreatorForPlan
	instanceStorage  storage.Instances
}

func NewInitialisationStep(os storage.Operations, is storage.Instances, b input.CreatorForPlan) *InitialisationStep {
	return &InitialisationStep{
		operationManager: process.NewOperationManager(os),
		inputBuilder:     b,
		instanceStorage:  is,
	}
}

func (s *InitialisationStep) Name() string {
	return "Provision_Initialization"
}

func (s *InitialisationStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	// create Provisioner InputCreator
	log.Infof("create provisioner input creator for %q plan ID", operation.ProvisioningParameters.PlanID)
	creator, err := s.inputBuilder.CreateProvisionInput(operation.ProvisioningParameters)

	switch {
	case err == nil:
		operation.InputCreator = creator
		err := s.updateInstance(operation.InstanceID, creator.Provider())
		if err != nil {
			return s.operationManager.RetryOperation(operation, "error while creating provisioning input creator", err, 1*time.Second, 5*time.Second, log)
		}

		return operation, 0, nil
	case kebError.IsTemporaryError(err):
		log.Errorf("cannot create input creator at the moment for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.RetryOperation(operation, "error while creating provisioning input creator", err, 5*time.Second, 5*time.Minute, log)
	default:
		log.Errorf("cannot create input creator for plan %s: %s", operation.ProvisioningParameters.PlanID, err)
		return s.operationManager.OperationFailed(operation, "cannot create provisioning input creator", err, log)
	}
}

func (s *InitialisationStep) updateInstance(id string, provider pkg.CloudProvider) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return fmt.Errorf("while getting instance: %w", err)
	}
	instance.Provider = provider
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return fmt.Errorf("while updating instance: %w", err)
	}

	return nil
}
