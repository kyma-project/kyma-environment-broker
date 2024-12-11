package suspension

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
)

type ContextUpdateHandler struct {
	operations          storage.Operations
	provisioningQueue   Adder
	deprovisioningQueue Adder

	log *slog.Logger
}

type Adder interface {
	Add(processId string)
}

func NewContextUpdateHandler(operations storage.Operations, provisioningQueue Adder, deprovisioningQueue Adder, l *slog.Logger) *ContextUpdateHandler {
	return &ContextUpdateHandler{
		operations:          operations,
		provisioningQueue:   provisioningQueue,
		deprovisioningQueue: deprovisioningQueue,
		log:                 l,
	}
}

// Handle performs suspension/unsuspension for given instance.
// Applies only when 'Active' parameter has changes and ServicePlanID is `Trial`
func (h *ContextUpdateHandler) Handle(instance *internal.Instance, newCtx internal.ERSContext) (bool, error) {
	l := h.log.With(
		"instanceID", instance.InstanceID,
		"runtimeID", instance.RuntimeID,
		"globalAccountID", instance.GlobalAccountID,
	)

	if !broker.IsTrialPlan(instance.ServicePlanID) {
		l.Info("Context update for non-trial instance, skipping")
		return false, nil
	}

	return h.handleContextChange(newCtx, instance, l)
}

func (h *ContextUpdateHandler) handleContextChange(newCtx internal.ERSContext, instance *internal.Instance, l *slog.Logger) (bool, error) {
	isActivated := true
	if instance.Parameters.ErsContext.Active != nil {
		isActivated = *instance.Parameters.ErsContext.Active
	}

	lastDeprovisioning, err := h.operations.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	// there was an error - fail
	if err != nil && !dberr.IsNotFound(err) {
		return false, err
	}

	if newCtx.Active == nil || isActivated == *newCtx.Active {
		l.Debug(fmt.Sprintf("Context.Active flag was not changed, the current value: %v", isActivated))
		if isActivated {
			// instance is marked as Active and incoming context update is unsuspension
			// TODO: consider retriggering failed unsuspension here
			l.Info(fmt.Sprintf("Context.Active flag is true - not triggering suspension for instance ID %s", instance.InstanceID))
			return false, nil
		}
		if !isActivated {
			// instance is inactive and incoming context update is suspension - verify if KEB should retrigger the operation
			if lastDeprovisioning.State == domain.Failed {
				l.Info(fmt.Sprintf("triggering suspension again for instance id %s", instance.InstanceID))
				return true, h.suspend(instance, l)
			}
			l.Info(fmt.Sprintf("last deprovisioning is not in Failed state - not triggering suspension for instance ID %s", instance.InstanceID))
			return false, nil
		}
	}

	if *newCtx.Active {
		if instance.IsExpired() {
			// if the instance is expired - do nothing
			return false, nil
		}
		if lastDeprovisioning != nil && !lastDeprovisioning.Temporary {
			l.Info(fmt.Sprintf("Instance has a deprovisioning operation %s (%s), skipping unsuspension.", lastDeprovisioning.ID, lastDeprovisioning.State))
			return false, nil
		}
		if lastDeprovisioning != nil && lastDeprovisioning.State == domain.Failed {
			err := fmt.Errorf("Preceding suspension has failed, unable to reliably unsuspend")
			return false, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "provisioning")
		}
		return true, h.unsuspend(instance, l)
	} else {
		return true, h.suspend(instance, l)
	}
}

func (h *ContextUpdateHandler) suspend(instance *internal.Instance, log *slog.Logger) error {
	lastDeprovisioning, err := h.operations.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	// there was an error - fail
	if err != nil && !dberr.IsNotFound(err) {
		return err
	}

	// no error, operation exists and is in progress
	if err == nil && (lastDeprovisioning.State == domain.InProgress || lastDeprovisioning.State == orchestration.Pending) {
		log.Info("Suspension already started")
		return nil
	}

	id := uuid.New().String()
	operation := internal.NewSuspensionOperationWithID(id, instance)
	err = h.operations.InsertDeprovisioningOperation(operation)
	if err != nil {
		return err
	}
	h.deprovisioningQueue.Add(operation.ID)
	return nil
}

func (h *ContextUpdateHandler) unsuspend(instance *internal.Instance, log *slog.Logger) error {
	if instance.IsExpired() {
		log.Info("Expired instance cannot be unsuspended")
		return nil
	}
	id := uuid.New().String()
	operation, err := internal.NewProvisioningOperationWithID(id, instance.InstanceID, instance.Parameters)
	operation.InstanceDetails, err = instance.GetInstanceDetails()

	// next suspension must set this flag once again
	operation.KimDeprovisionsOnly = nil

	if err != nil {
		h.log.Error(fmt.Sprintf("unable to extract shoot name: %s", err.Error()))
		return err
	}
	operation.State = orchestration.Pending
	log.Info(fmt.Sprintf("Starting unsuspension: shootName=%s shootDomain=%s", operation.ShootName, operation.ShootDomain))
	// RuntimeID must be cleaned  - this mean that there is no runtime in the provisioner/director
	operation.RuntimeID = ""
	operation.DashboardURL = instance.DashboardURL

	err = h.operations.InsertProvisioningOperation(operation)
	if err != nil {
		return err
	}
	h.provisioningQueue.Add(operation.ID)
	return nil
}
