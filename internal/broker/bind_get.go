package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
)

type GetBindingEndpoint struct {
	log      logrus.FieldLogger
	bindings storage.Bindings
}

func NewGetBinding(log logrus.FieldLogger, bindings storage.Bindings) *GetBindingEndpoint {
	return &GetBindingEndpoint{log: log.WithField("service", "GetBindingEndpoint"), bindings: bindings}
}

// GetBinding fetches an existing service binding
//
//	GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *GetBindingEndpoint) GetBinding(_ context.Context, instanceID, bindingID string, _ domain.FetchBindingDetails) (domain.GetBindingSpec, error) {
	b.log.Infof("GetBinding instanceID: %s", instanceID)
	b.log.Infof("GetBinding bindingID: %s", bindingID)

	binding, err := b.bindings.Get(instanceID, bindingID)

	if binding == nil {
		message := "Binding not found"
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
