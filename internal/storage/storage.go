package storage

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gocraft/dbr"
	eventsapi "github.com/kyma-project/kyma-environment-broker/common/events"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/driver/memory"
	postgres "github.com/kyma-project/kyma-environment-broker/internal/storage/driver/postsql"
	eventstorage "github.com/kyma-project/kyma-environment-broker/internal/storage/driver/postsql/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
)

type BrokerStorage interface {
	Instances() Instances
	Operations() Operations
	Provisioning() Provisioning
	Deprovisioning() Deprovisioning
	SubaccountStates() SubaccountStates
	Events() Events
	InstancesArchived() InstancesArchived
	Bindings() Bindings
	Actions() Actions
	TimeZones() TimeZones
	EncryptionModeStats() EncryptionModeStats
}

const (
	connectionRetries = 10
)

func NewFromConfig(cfg Config, evcfg events.Config, cipher postgres.Cipher) (BrokerStorage, *dbr.Connection, error) {
	return NewFromConfigAndConnectionURL(cfg, evcfg, cipher, cfg.ConnectionURL())
}

func NewFromConfigAndConnectionURL(cfg Config, evcfg events.Config, cipher postgres.Cipher, connectionURL string) (BrokerStorage, *dbr.Connection, error) {
	slog.Info(fmt.Sprintf("Setting DB connection pool params: connectionMaxLifetime=%s maxIdleConnections=%d maxOpenConnections=%d",
		cfg.ConnMaxLifetime, cfg.MaxIdleConns, cfg.MaxOpenConns))

	connection, err := postsql.InitializeDatabase(connectionURL, connectionRetries)
	if err != nil {
		return nil, nil, err
	}

	connection.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	connection.SetMaxIdleConns(cfg.MaxIdleConns)
	connection.SetMaxOpenConns(cfg.MaxOpenConns)

	factory := postsql.NewFactory(connection)

	operation := postgres.NewOperation(factory, cipher)
	return storage{
		instance:            postgres.NewInstance(factory, operation, cipher),
		operation:           operation,
		events:              events.New(evcfg, eventstorage.New(factory)),
		subaccountStates:    postgres.NewSubaccountStates(factory),
		instancesArchived:   postgres.NewInstanceArchived(factory),
		bindings:            postgres.NewBinding(factory, cipher),
		actions:             postgres.NewAction(factory),
		timezones:           postgres.NewTimeZones(factory),
		encryptionModeStats: postgres.NewEncryptionModeStats(factory),
	}, connection, nil
}

func NewMemoryStorage() BrokerStorage {
	op := memory.NewOperation()
	ss := memory.NewSubaccountStates()
	return storage{
		operation:         op,
		subaccountStates:  ss,
		instance:          memory.NewInstance(op, ss),
		events:            events.New(events.Config{}, NewInMemoryEvents()),
		instancesArchived: memory.NewInstanceArchivedInMemoryStorage(),
		bindings:          memory.NewBinding(),
		actions:           memory.NewAction(),
	}
}

type inMemoryEvents struct {
	events []eventsapi.EventDTO
}

func NewInMemoryEvents() *inMemoryEvents {
	return &inMemoryEvents{
		events: make([]eventsapi.EventDTO, 0),
	}
}

func (_ inMemoryEvents) RunGarbageCollection(pollingPeriod, retention time.Duration) {
	return
}

func (e *inMemoryEvents) InsertEvent(eventLevel eventsapi.EventLevel, message, instanceID, operationID string) {
	e.events = append(e.events, eventsapi.EventDTO{Level: eventLevel, InstanceID: &instanceID, OperationID: &operationID, Message: message})
	slog.Info(fmt.Sprintf("EVENT [instanceID=%v/operationID=%v] %v: %v", instanceID, operationID, eventLevel, message))
}

func (e *inMemoryEvents) ListEvents(filter eventsapi.EventFilter) ([]eventsapi.EventDTO, error) {
	var events []eventsapi.EventDTO
	for _, ev := range e.events {
		if !requiredContains(ev.InstanceID, filter.InstanceIDs) {
			continue
		}
		if !requiredContains(ev.OperationID, filter.OperationIDs) {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

func requiredContains[T comparable](el *T, sl []T) bool {
	if len(sl) == 0 {
		return true
	}
	if el == nil {
		return false
	}
	for _, x := range sl {
		if *el == x {
			return true
		}
	}
	return false
}

type storage struct {
	instance            Instances
	operation           Operations
	events              Events
	subaccountStates    SubaccountStates
	instancesArchived   InstancesArchived
	bindings            Bindings
	actions             Actions
	timezones           TimeZones
	encryptionModeStats EncryptionModeStats
}

func (s storage) Instances() Instances {
	return s.instance
}

func (s storage) Operations() Operations {
	return s.operation
}

func (s storage) Provisioning() Provisioning {
	return s.operation
}

func (s storage) Deprovisioning() Deprovisioning {
	return s.operation
}

func (s storage) Events() Events {
	return s.events
}

func (s storage) SubaccountStates() SubaccountStates {
	return s.subaccountStates
}

func (s storage) InstancesArchived() InstancesArchived {
	return s.instancesArchived
}

func (s storage) Bindings() Bindings {
	return s.bindings
}

func (s storage) Actions() Actions {
	return s.actions
}

func (s storage) TimeZones() TimeZones { return s.timezones }

func (s storage) EncryptionModeStats() EncryptionModeStats {
	return s.encryptionModeStats
}
