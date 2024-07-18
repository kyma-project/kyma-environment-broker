package provisioning

import (
	"fmt"

	"github.com/google/uuid"

	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/sirupsen/logrus"
)

type GenerateRuntimeIDStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
}

func NewGenerateRuntimeIDStep(os storage.Operations, is storage.Instances) *GenerateRuntimeIDStep {
	return &GenerateRuntimeIDStep{
		operationManager: process.NewOperationManager(os),
		instanceStorage:  is,
	}
}

func (s *GenerateRuntimeIDStep) Name() string {
	return "Generate_Runtime_ID"
}

func (s *GenerateRuntimeIDStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID != "" {
		log.Infof("RuntimeID already set %s, skipping", operation.RuntimeID)
		return operation, 0, nil
	}

	runtimeID := uuid.New().String()

	log.Infof("RuntimeID %s generated", runtimeID)

	repeatAfter := time.Duration(0)
	operation, repeatAfter, _ = s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.RuntimeID = runtimeID
		operation.ProvisionerOperationID = ""
	}, log)
	if repeatAfter != 0 {
		log.Errorf("cannot save RuntimeID in operation")
		return operation, 5 * time.Second, nil
	}

	err := s.updateInstance(operation.InstanceID, runtimeID)

	switch {
	case err == nil:
	case dberr.IsConflict(err):
		err := s.updateInstance(operation.InstanceID, runtimeID)
		if err != nil {
			log.Errorf("cannot update instance: %s", err)
			return operation, 1 * time.Minute, nil
		}
	}

	return operation, 0, nil
}

func (s *GenerateRuntimeIDStep) updateInstance(id, runtimeID string) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return fmt.Errorf("while getting instance: %w", err)
	}
	instance.RuntimeID = runtimeID
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return fmt.Errorf("while updating instance: %w", err)
	}

	return nil
}
