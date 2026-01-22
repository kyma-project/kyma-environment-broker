package cis

import (
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

//go:generate mockery --name=CisClient --output=automock
type CisClient interface {
	FetchSubaccountsToDelete() ([]string, error)
}

//go:generate mockery --name=BrokerClient --output=automock
type BrokerClient interface {
	Deprovision(instance internal.Instance) (string, error)
}

type SubAccountCleanupService struct {
	client       CisClient
	brokerClient BrokerClient
	storage      storage.Instances
	chunksAmount int
	log          *slog.Logger

	deprovisioned atomic.Int64
	failed        atomic.Int64
}

func NewSubAccountCleanupService(client CisClient, brokerClient BrokerClient, storage storage.Instances, log *slog.Logger) *SubAccountCleanupService {
	return &SubAccountCleanupService{
		client:       client,
		brokerClient: brokerClient,
		storage:      storage,
		chunksAmount: 50,
		log:          log,
	}
}

func (ac *SubAccountCleanupService) Run() error {
	ac.log.Info("Starting SubAccount cleanup process")

	subaccounts, err := ac.client.FetchSubaccountsToDelete()
	if err != nil {
		return fmt.Errorf("while fetching subaccounts by client: %w", err)
	}

	subaccountsBatch := chunk(ac.chunksAmount, subaccounts)
	chunks := len(subaccountsBatch)
	ac.log.Info("Subaccounts divided into chunks", "chunks", chunks)

	errCh := make(chan error)
	done := make(chan struct{})
	var isDone bool

	for i, chunk := range subaccountsBatch {
		ac.log.Info(
			"Starting deprovisioning goroutine",
			"chunkIndex", i,
			"subaccountsInChunk", len(chunk),
		)
		go ac.executeDeprovisioning(chunk, done, errCh)
	}

	for !isDone {
		select {
		case err := <-errCh:
			slog.Warn(fmt.Sprintf("Part of deprovisioning process failed: %s", err))
		case <-done:
			chunks--
			ac.log.Info("Deprovisioning chunk finished", "remainingChunks", chunks)
			if chunks == 0 {
				isDone = true
			}
		}
	}

	slog.Info(
		"SubAccount cleanup process finished",
		"instancesDeprovisioned", ac.deprovisioned.Load(),
		"instancesFailed", ac.failed.Load(),
	)
	return nil
}

func (ac *SubAccountCleanupService) executeDeprovisioning(subaccounts []string, done chan<- struct{}, errCh chan<- error) {
	instances, err := ac.storage.FindAllInstancesForSubAccounts(subaccounts)
	if err != nil {
		errCh <- fmt.Errorf("while finding all instances by subaccounts: %w", err)
		return
	}

	for _, instance := range instances {
		operation, err := ac.brokerClient.Deprovision(instance)
		if err != nil {
			ac.failed.Add(1)
			errCh <- fmt.Errorf("while deprovisioning instance with ID %s: %w", instance.InstanceID, err)
			continue
		}
		ac.deprovisioned.Add(1)
		ac.log.Info(
			"Deprovisioning triggered",
			"subAccountID", instance.SubAccountID,
			"instanceID", instance.InstanceID,
			"operation", operation,
		)
	}

	done <- struct{}{}
}

func chunk(amount int, data []string) [][]string {
	var divided [][]string

	for i := 0; i < len(data); i += amount {
		end := i + amount
		if end > len(data) {
			end = len(data)
		}
		divided = append(divided, data[i:end])
	}

	return divided
}
