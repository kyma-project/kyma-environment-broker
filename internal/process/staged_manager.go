package process

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/pkg/errors"

	"github.com/kyma-project/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/ans-manager/ans_manager"
	"github.com/kyma-project/ans-manager/events"
	"github.com/kyma-project/ans-manager/notifications"

	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type StagedManager struct {
	log              *slog.Logger
	operationStorage storage.Operations
	publisher        event.Publisher

	stages           []*stage
	operationTimeout time.Duration

	mu sync.RWMutex

	speedFactor         int64
	cfg                 StagedManagerConfiguration
	notificationService *ans.Service
}

type StagedManagerConfiguration struct {
	// Max time of processing a step by a worker without returning to the queue
	MaxStepProcessingTime time.Duration `envconfig:"default=2m"`
	WorkersAmount         int           `envconfig:"default=20"`
}

func (c StagedManagerConfiguration) String() string {
	return fmt.Sprintf("(MaxStepProcessingTime=%s; WorkersAmount=%d)", c.MaxStepProcessingTime, c.WorkersAmount)
}

type Step interface {
	Name() string
	Run(operation internal.Operation, logger *slog.Logger) (internal.Operation, time.Duration, error)
}

type StepCondition func(operation internal.Operation) bool

type StepWithCondition struct {
	Step
	condition StepCondition
}

type stage struct {
	name  string
	steps []StepWithCondition
}

func (s *stage) AddStep(step Step, cnd StepCondition) {
	s.steps = append(s.steps, StepWithCondition{
		Step:      step,
		condition: cnd,
	})
}

func NewStagedManager(storage storage.Operations, pub event.Publisher, operationTimeout time.Duration, cfg StagedManagerConfiguration, service *ans.Service, logger *slog.Logger) *StagedManager {
	return &StagedManager{
		log:                 logger,
		operationStorage:    storage,
		publisher:           pub,
		operationTimeout:    operationTimeout,
		speedFactor:         1,
		cfg:                 cfg,
		notificationService: service,
	}
}

// SpeedUp changes speedFactor parameter to reduce the sleep time if a step needs a retry.
// This method should only be used for testing purposes
func (m *StagedManager) SpeedUp(speedFactor int64) {
	m.speedFactor = speedFactor
}

func (m *StagedManager) DefineStages(names []string) {
	m.stages = make([]*stage, len(names))
	for i, n := range names {
		m.stages[i] = &stage{name: n, steps: []StepWithCondition{}}
	}
}

func (m *StagedManager) AddStep(stageName string, step Step, cnd StepCondition) error {
	for _, s := range m.stages {
		if s.name == stageName {
			s.AddStep(step, cnd)
			return nil
		}
	}
	return fmt.Errorf("stage %s not defined", stageName)
}

func (m *StagedManager) GetAllStages() []string {
	var all []string
	for _, s := range m.stages {
		all = append(all, s.name)
	}
	return all
}

func (m *StagedManager) Execute(operationID string) (time.Duration, error) {

	operation, err := m.operationStorage.GetOperationByID(operationID)
	if err != nil {
		m.log.Error(fmt.Sprintf("Cannot fetch operation from storage: %s", err))
		return 3 * time.Second, nil
	}

	logOperation := m.log.With("operationID", operationID, "instanceID", operation.InstanceID, "planID", operation.ProvisioningParameters.PlanID)
	logOperation.Info(fmt.Sprintf("Start process operation steps for GlobalAccount=%s, ", operation.ProvisioningParameters.ErsContext.GlobalAccountID))
	if time.Since(operation.CreatedAt) > m.operationTimeout {
		timeoutErr := kebError.TimeoutError("operation has reached the time limit", string(kebError.KEBDependency))
		operation.LastError = timeoutErr
		defer m.publishEventOnFail(operation, err)
		logOperation.Info(fmt.Sprintf("operation has reached the time limit: operation was created at: %s", operation.CreatedAt))
		operation.State = domain.Failed
		_, err = m.operationStorage.UpdateOperation(*operation)
		if err != nil {
			logOperation.Info("Unable to save operation with finished the provisioning process")
			timeoutErr = timeoutErr.SetMessage(fmt.Sprintf("%s and %s", timeoutErr.Error(), err.Error()))
			operation.LastError = timeoutErr
			return time.Second, timeoutErr
		}

		return 0, timeoutErr
	}

	var when time.Duration
	processedOperation := *operation

	for _, stage := range m.stages {
		if processedOperation.IsStageFinished(stage.name) {
			continue
		}

		for _, step := range stage.steps {
			logStep := logOperation.With("step", step.Name()).
				With("stage", stage.name)
			if step.condition != nil && !step.condition(processedOperation) {
				logStep.Debug("Skipping")
				continue
			}
			operation.EventInfof("processing step: %v", step.Name())
			err = m.sendResourceEventForStep(&processedOperation, step.Name(), logStep)
			if err != nil {
				logOperation := logStep.With("step", step.Name(), "operationID", processedOperation.ID)
				logOperation.Error(fmt.Sprintf("Failed to send resource event for step %s: %s", step.Name(), err))
			}

			processedOperation, when, err = m.runStep(step, processedOperation, logStep)
			if err != nil {
				logStep.Error(fmt.Sprintf("Process operation failed: %s", err))
				operation.EventErrorf(err, "step %v processing returned error", step.Name())
				return 0, err
			}
			if processedOperation.State == domain.Failed || processedOperation.State == domain.Succeeded {
				logStep.Info(fmt.Sprintf("Operation %q got status %s. Process finished.", operation.ID, processedOperation.State))
				operation.EventInfof("operation processing %v", processedOperation.State)
				m.publishOperationFinishedEvent(processedOperation)
				m.publishDeprovisioningSucceeded(&processedOperation)
				return 0, nil
			}

			// the step needs a retry
			if when > 0 {
				logStep.Warn(fmt.Sprintf("retrying step %s by restarting the operation in %d s", step.Name(), int64(when.Seconds())))
				return when, nil
			}
			logStep.Info(fmt.Sprintf("Step %q processed successfully", step.Name()))
		}

		processedOperation, err = m.saveFinishedStage(processedOperation, stage, logOperation)

		// it is ok, when operation does not exist in the DB - it can happen at the end of a deprovisioning process
		if err != nil && !dberr.IsNotFound(err) {
			return time.Second, nil
		}
	}

	logOperation.Info("Operation succeeded")

	processedOperation.State = domain.Succeeded
	processedOperation.Description = "Processing finished"

	m.publishEventOnSuccess(&processedOperation)

	_, err = m.operationStorage.UpdateOperation(processedOperation)
	// it is ok, when operation does not exist in the DB - it can happen at the end of a deprovisioning process
	if err != nil && !dberr.IsNotFound(err) {
		logOperation.Info("Unable to save operation with finished the provisioning process")
		return time.Second, err
	}

	return 0, nil
}

func (m *StagedManager) saveFinishedStage(operation internal.Operation, s *stage, log *slog.Logger) (internal.Operation, error) {
	operation.FinishStage(s.name)
	op, err := m.operationStorage.UpdateOperation(operation)
	// it is ok, when operation does not exist in the DB - it can happen at the end of a deprovisioning process
	if err != nil && !dberr.IsNotFound(err) {
		log.Info(fmt.Sprintf("Unable to save operation with finished stage %s: %s", s.name, err.Error()))
		return operation, err
	}
	log.Info(fmt.Sprintf("Finished stage %s", s.name))
	return *op, nil
}

func (m *StagedManager) runStep(step Step, operation internal.Operation, logger *slog.Logger) (processedOperation internal.Operation, backoff time.Duration, err error) {
	var start time.Time
	defer func() {
		if pErr := recover(); pErr != nil {
			logger.Info(fmt.Sprintf("panic in RunStep in staged manager: %v", pErr))
			err = errors.New(fmt.Sprintf("%v", pErr))
			om := NewOperationManager(m.operationStorage, step.Name(), kebError.KEBDependency)
			processedOperation, _, _ = om.OperationFailed(operation, "recovered from panic", err, m.log)
		}
	}()

	processedOperation = operation
	begin := time.Now()
	for {
		start = time.Now()
		logger.Info("Start step")
		stepLogger := logger.With("step", step.Name(), "operationID", processedOperation.ID)
		processedOperation, backoff, err = step.Run(processedOperation, stepLogger)
		if err != nil {
			logOperation := stepLogger.With("error_component", processedOperation.LastError.GetComponent(), "error_reason", processedOperation.LastError.GetReason())
			logOperation.Warn(fmt.Sprintf("Last error from step: %s", processedOperation.LastError.Error()))
			// only save to storage, skip for alerting if error
			_, err = m.operationStorage.UpdateOperation(processedOperation)
			if err != nil {
				logOperation.Error("unable to save operation with resolved last error from step, additionally, see previous logs for ealier errors")
			}
		}

		m.publisher.Publish(context.TODO(), OperationStepProcessed{
			StepProcessed: StepProcessed{
				StepName: step.Name(),
				Duration: time.Since(start),
				When:     backoff,
				Error:    err,
			},
			Operation:    processedOperation,
			OldOperation: operation,
		})

		// break the loop if:
		// - the step does not need a retry
		// - step returns an error
		// - the loop takes too much time (to not block the worker too long)
		if backoff == 0 || err != nil || time.Since(begin) > m.cfg.MaxStepProcessingTime {
			if err != nil {
				logOperation := m.log.With("step", step.Name(), "operationID", processedOperation.ID, "error_component", processedOperation.LastError.GetComponent(), "error_reason", processedOperation.LastError.GetReason())
				logOperation.Error(fmt.Sprintf("Last Error that terminated the step: %s", processedOperation.LastError.Error()))
			}
			return processedOperation, backoff, err
		}
		operation.EventInfof("step %v sleeping for %v", step.Name(), backoff)
		time.Sleep(backoff / time.Duration(m.speedFactor))
	}
}

func (m *StagedManager) publishEventOnFail(operation *internal.Operation, err error) {
	logOperation := m.log.With("operationID", operation.ID, "error_component", operation.LastError.GetComponent(), "error_reason", operation.LastError.GetReason())
	logOperation.Error(fmt.Sprintf("Last error: %s", operation.LastError.Error()))

	m.publishOperationFinishedEvent(*operation)

	m.publisher.Publish(context.TODO(), OperationStepProcessed{
		StepProcessed: StepProcessed{
			Duration: time.Since(operation.CreatedAt),
			Error:    err,
		},
		OldOperation: *operation,
		Operation:    *operation,
	})
}

func (m *StagedManager) publishEventOnSuccess(operation *internal.Operation) {
	m.publisher.Publish(context.TODO(), OperationSucceeded{
		Operation: *operation,
	})

	m.publishOperationFinishedEvent(*operation)

	m.publishDeprovisioningSucceeded(operation)
}

func (m *StagedManager) publishOperationFinishedEvent(operation internal.Operation) {
	m.publisher.Publish(context.TODO(), OperationFinished{
		Operation: operation,
		PlanID:    broker.PlanID(operation.ProvisioningParameters.PlanID),
	})
}

func (m *StagedManager) publishDeprovisioningSucceeded(operation *internal.Operation) {
	if operation.State == domain.Succeeded && operation.Type == internal.OperationTypeDeprovision {
		m.publisher.Publish(
			context.TODO(), DeprovisioningSucceeded{
				Operation: internal.DeprovisioningOperation{Operation: *operation},
			},
		)
	}
}

func (m *StagedManager) sendResourceEventForStep(operation *internal.Operation, stepName string, logger *slog.Logger) error {
	if m.notificationService != nil {
		logger.Info("Sending resource event to ANS")

		event, err := events.NewResourceEvent(
			"KEB:step-event",
			stepName,
			fmt.Sprintf("%s:%s", operation.ShootName, operation.Type),
			events.NewResource("broker",
				"keb",
				"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad",
				"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad",
				events.WithResourceGlobalAccount("8cd57dc2-edb2-45e0-af8b-7d881006e516")),
			events.SeverityInfo,
			events.CategoryNotification,
			events.VisibilityOwnerSubAccount,
			*events.NewNotificationMapping("POC_WebOnlyType4",
				*events.NewRecipients(
					[]events.XsuaaRecipient{*events.NewXsuaaRecipient(events.LevelSubaccount, "2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad", []events.RoleName{"Subaccount admin"})},
					nil)),
		)
		if err != nil {
			logger.Error(fmt.Sprintf("cannot create event: %s", err))
			return fmt.Errorf("cannot create event")
		}
		err = m.notificationService.PostEvent(*event)
		if err != nil {
			logger.Error("Failed to post event to ANS", "error", err)
			return fmt.Errorf("failed to post event to ANS: %w", err)
		} else {
			logger.Info("Event posted to ANS successfully")
		}
	}
	return nil
}
