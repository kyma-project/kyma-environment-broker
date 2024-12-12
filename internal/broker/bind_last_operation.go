package broker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pivotal-cf/brokerapi/v8/domain"
)

type LastBindingOperationEndpoint struct {
	log *slog.Logger
}

func NewLastBindingOperation(log *slog.Logger) *LastBindingOperationEndpoint {
	return &LastBindingOperationEndpoint{log: log.With("service", "LastBindingOperationEndpoint")}
}

// LastBindingOperation fetches last operation state for a service binding
//
//	GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation
func (b *LastBindingOperationEndpoint) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	b.log.Info(fmt.Sprintf("LastBindingOperation instanceID: %s", instanceID))
	b.log.Info(fmt.Sprintf("LastBindingOperation bindingID: %s", bindingID))
	b.log.Info(fmt.Sprintf("LastBindingOperation details: %+v", details))

	return domain.LastOperation{}, fmt.Errorf("not supported")
}
