package fixture

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type BindingOption func(binding *internal.Binding)

func WithOffset(offset time.Duration) BindingOption {
	return func(b *internal.Binding) {
		b.CreatedAt = time.Now().Add(-offset)
		b.UpdatedAt = time.Now().Add(time.Minute*5 - offset)
		b.ExpiresAt = time.Now().Add(time.Minute*10 - offset)
	}
}

func WithInstanceID(instanceID string) BindingOption {
	return func(b *internal.Binding) {
		b.InstanceID = instanceID
	}
}

func FixBinding(id string, opts ...BindingOption) internal.Binding {
	binding := internal.Binding{
		ID:         id,
		InstanceID: fmt.Sprintf("instance-%s", id),

		CreatedAt: time.Now(),
		UpdatedAt: time.Now().Add(time.Minute * 5),
		ExpiresAt: time.Now().Add(time.Minute * 10),

		Kubeconfig:        "kubeconfig",
		ExpirationSeconds: 600,
		CreatedBy:         "john.smith@email.com",
	}

	for _, opt := range opts {
		opt(&binding)
	}
	return binding
}
