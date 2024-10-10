package broker

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type UnbindEndpoint struct {
	log             logrus.FieldLogger
	bindingsStorage storage.Bindings
}

func NewUnbind(log logrus.FieldLogger, bindingsStorage storage.Bindings) *UnbindEndpoint {
	return &UnbindEndpoint{log: log.WithField("service", "UnbindEndpoint"), bindingsStorage: bindingsStorage}
}

// Unbind deletes an existing service binding
//
//	DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}
func (b *UnbindEndpoint) Unbind(ctx context.Context, instanceID, bindingID string, details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {
	b.log.Infof("Unbind instanceID: %s", instanceID)
	b.log.Infof("Unbind details: %+v", details)
	b.log.Infof("Unbind asyncAllowed: %v", asyncAllowed)

	err := b.bindingsStorage.Delete(instanceID, bindingID)

	if err != nil {
		b.log.Errorf("Unbind error: %s", err)
		return domain.UnbindSpec{}, err
	}

	return domain.UnbindSpec{
		IsAsync: false,
	}, nil
}
