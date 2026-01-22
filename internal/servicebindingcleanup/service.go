package servicebindingcleanup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type BrokerClient interface {
	Unbind(binding internal.Binding) error
}

type Service struct {
	dryRun          bool
	brokerClient    BrokerClient
	bindingsStorage storage.Bindings
}

func NewService(dryRun bool, client BrokerClient, bindingsStorage storage.Bindings) *Service {
	return &Service{
		dryRun:          dryRun,
		brokerClient:    client,
		bindingsStorage: bindingsStorage,
	}
}

func (s *Service) PerformCleanup() error {
	slog.Info(fmt.Sprintf("Fetching Service Bindings with expires_at <= %q", time.Now().UTC().Truncate(time.Second).String()))
	bindings, err := s.bindingsStorage.ListExpired()
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Expired Service Bindings: %d", len(bindings)))
	if s.dryRun {
		return nil
	}
	slog.Info("Requesting Service Bindings removal...")
	for _, binding := range bindings {
		if err := s.brokerClient.Unbind(binding); err != nil {
			var unexpectedStatusCodeErr broker.UnexpectedStatusCodeError
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if errors.As(err, &unexpectedStatusCodeErr) && unexpectedStatusCodeErr.UnexpectedStatusCode == http.StatusGone {
				slog.Info(fmt.Sprintf("instance with ID: %q does not exist for service binding with ID %q", binding.InstanceID, binding.ID))
				continue
			}
			return fmt.Errorf("while sending unbind request for service binding ID %q and instance ID %q: %w", binding.ID, binding.InstanceID, err)
		}
	}
	return nil
}
