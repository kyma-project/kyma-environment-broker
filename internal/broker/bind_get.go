package broker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
)

type GetBindingEndpoint struct {
	log        logrus.FieldLogger
	bindings   storage.Bindings
	operations storage.Operations
}

func NewGetBinding(log logrus.FieldLogger, db storage.BrokerStorage) *GetBindingEndpoint {
	return &GetBindingEndpoint{log: log.WithField("service", "GetBindingEndpoint"), bindings: db.Bindings(), operations: db.Operations()}
}

// GetBinding fetches an existing service binding
//
//	GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *GetBindingEndpoint) GetBinding(_ context.Context, instanceID, bindingID string, _ domain.FetchBindingDetails) (domain.GetBindingSpec, error) {
	b.log.Infof("GetBinding instanceID: %s", instanceID)
	b.log.Infof("GetBinding bindingID: %s", bindingID)

	lastOperation, err := b.operations.GetLastOperation(instanceID)
	if err != nil {
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get last operation for instance %s", instanceID), http.StatusInternalServerError, fmt.Sprintf("failed to get last operation %s", instanceID))
	}
	if lastOperation.Type == internal.OperationTypeDeprovision {
		message := "Binding not found"
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusNotFound, message)
	}

	binding, err := b.bindings.Get(instanceID, bindingID)

	if binding == nil {
		message := "Binding not found"
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusNotFound, message)
	}

	if binding.ExpiresAt.Before(time.Now()) {
		b.log.Infof("GetBinding was called for expired binding %s for instance %s", bindingID, instanceID)
		message := "Binding expired"
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusNotFound, message)
	}

	if len(binding.Kubeconfig) == 0 {
		message := "Binding creation in progress"
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusNotFound, message)
	}

	if err != nil {
		b.log.Errorf("GetBinding error: %s", err)
		message := fmt.Sprintf("Unexpected error: %s", err)
		return domain.GetBindingSpec{}, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusInternalServerError, message)
	}

	return domain.GetBindingSpec{
		Credentials: Credentials{
			Kubeconfig: binding.Kubeconfig,
		},
	}, nil
}
