package servicebindingcleanup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
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
	} else {
		slog.Info("Requesting Service Bindings removal...")
		for _, binding := range bindings {
			err := s.brokerClient.Unbind(binding)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				slog.Error(fmt.Sprintf("while sending unbind request for service binding ID %q: %s", binding.ID, err))
				break
			}
		}
	}
	return nil
}
