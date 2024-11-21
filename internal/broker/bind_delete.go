package broker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/event"

	"github.com/kyma-project/kyma-environment-broker/internal"
	broker "github.com/kyma-project/kyma-environment-broker/internal/broker/bindings"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
)

type UnbindEndpoint struct {
	log               logrus.FieldLogger
	bindingsStorage   storage.Bindings
	instancesStorage  storage.Instances
	operationsStorage storage.Operations
	bindingsManager   broker.BindingsManager
	publisher         event.Publisher
}

func NewUnbind(log logrus.FieldLogger, db storage.BrokerStorage, bindingsManager broker.BindingsManager, publisher event.Publisher) *UnbindEndpoint {
	return &UnbindEndpoint{log: log.WithField("service", "UnbindEndpoint"),
		bindingsStorage:   db.Bindings(),
		instancesStorage:  db.Instances(),
		bindingsManager:   bindingsManager,
		operationsStorage: db.Operations(),
		publisher:         publisher,
	}
}

// Unbind deletes an existing service binding
//
//	DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *UnbindEndpoint) Unbind(ctx context.Context, instanceID, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {
	start := time.Now()
	response, err := b.unbind(ctx, instanceID, bindingID, details, asyncAllowed)
	processingDuration := time.Since(start)

	b.publisher.Publish(ctx, UnbindRequestProcessed{ProcessingDuration: processingDuration, Error: err})
	return response, err
}

func (b *UnbindEndpoint) unbind(ctx context.Context, instanceID, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {

	b.log.Infof("Unbind instanceID: %s", instanceID)
	b.log.Infof("Unbind details: %+v", details)
	b.log.Infof("Unbind asyncAllowed: %v", asyncAllowed)

	instance, err := b.instancesStorage.GetByID(instanceID)
	switch {
	case dberr.IsNotFound(err):
		err = b.bindingsStorage.Delete(instanceID, bindingID)
		if err != nil {
			b.log.Errorf("Unbind error during removal of db entity: %v", err)
			return domain.UnbindSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to delete binding for binding %s and not existing instance %s: %v", bindingID, instanceID, err), http.StatusInternalServerError, fmt.Sprintf("failed to delete resources for binding %s and not existing instance %s: %v", bindingID, instanceID, err))
		}
		return domain.UnbindSpec{}, apiresponses.ErrInstanceDoesNotExist
	case err != nil:
		return domain.UnbindSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get instance %s", instanceID), http.StatusInternalServerError, fmt.Sprintf("failed to get instance %s", instanceID))
	}

	_, err = b.bindingsStorage.Get(instanceID, bindingID)
	switch {
	case dberr.IsNotFound(err):
		return domain.UnbindSpec{}, apiresponses.ErrBindingDoesNotExist
	case err != nil:
		return domain.UnbindSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get instance %s", instanceID), http.StatusInternalServerError, fmt.Sprintf("failed to get instance %s", instanceID))
	}

	lastOperation, err := b.operationsStorage.GetLastOperation(instance.InstanceID)
	if err != nil {
		return domain.UnbindSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get last operation for instance %s", instanceID), http.StatusInternalServerError, fmt.Sprintf("failed to get last operation %s", instanceID))
	}

	if lastOperation.Type != internal.OperationTypeDeprovision {
		err = b.bindingsManager.Delete(ctx, instance, bindingID)
		if err != nil {
			b.log.Errorf("Unbind error during removal of service account resources: %s", err)
			return domain.UnbindSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to delete binding resources for binding %s and instance %s: %v", bindingID, instanceID, err), http.StatusInternalServerError, fmt.Sprintf("failed to delete resources for binding %s and instance %s: %v", bindingID, instanceID, err))
		}
	}

	err = b.bindingsStorage.Delete(instanceID, bindingID)
	if err != nil {
		b.log.Errorf("Unbind error during removal of db entity: %v", err)
		return domain.UnbindSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to delete binding resources for binding %s and instance %s: %v", bindingID, instanceID, err), http.StatusInternalServerError, fmt.Sprintf("failed to delete resources for binding %s and instance %s: %v", bindingID, instanceID, err))
	}
	b.log.Infof("Successfully removed binding %s for instance %s", bindingID, instanceID)

	return domain.UnbindSpec{
		IsAsync: false,
	}, nil
}
