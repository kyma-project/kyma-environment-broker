package postsql

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/events"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/predicate"

	"github.com/gocraft/dbr"
)

//go:generate mockery --name=Factory
type Factory interface {
	NewReadSession() ReadSession
	NewWriteSession() WriteSession
	NewSessionWithinTransaction() (WriteSessionWithinTransaction, dberr.Error)
}

//go:generate mockery --name=ReadSession
type ReadSession interface {
	FindAllInstancesJoinedWithOperation(prct ...predicate.Predicate) ([]dbmodel.InstanceWithOperationDTO, dberr.Error)
	FindAllInstancesForRuntimes(runtimeIdList []string) ([]dbmodel.InstanceDTO, dberr.Error)
	FindAllInstancesForSubAccounts(subAccountslist []string) ([]dbmodel.InstanceDTO, dberr.Error)
	GetInstanceByID(instanceID string) (dbmodel.InstanceDTO, dberr.Error)
	GetLastOperation(instanceID string, types []internal.OperationType) (dbmodel.OperationDTO, dberr.Error)
	GetOperationByID(opID string) (dbmodel.OperationDTO, dberr.Error)
	GetNotFinishedOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	CountNotFinishedOperationsByInstanceID(instanceID string) (int, dberr.Error)
	GetOperationByTypeAndInstanceID(inID string, opType internal.OperationType) (dbmodel.OperationDTO, dberr.Error)
	GetOperationByInstanceID(inID string) (dbmodel.OperationDTO, dberr.Error)
	GetOperationsByTypeAndInstanceID(inID string, opType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationsByInstanceID(inID string) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationsForIDs(opIdList []string) ([]dbmodel.OperationDTO, dberr.Error)
	ListOperations(filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error)
	ListOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error)
	GetOperationStats() ([]dbmodel.OperationStatEntry, error)
	GetActiveInstanceStats() ([]dbmodel.InstanceByGlobalAccountIDStatEntry, error)
	GetSubaccountsInstanceStats() ([]dbmodel.InstanceBySubAccountIDStatEntry, error)
	GetERSContextStats() ([]dbmodel.InstanceERSContextStatsEntry, error)
	GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error)
	ListInstances(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceWithExtendedOperationDTO, int, int, error)
	ListInstancesWithSubaccountStates(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceWithSubaccountStateDTO, int, int, error)
	ListOperationsInTimeRange(from, to time.Time) ([]dbmodel.OperationDTO, error)
	ListEvents(filter events.EventFilter) ([]events.EventDTO, error)
	GetDistinctSubAccounts() ([]string, dberr.Error)
	ListSubaccountStates() ([]dbmodel.SubaccountStateDTO, dberr.Error)
	GetInstanceArchivedByID(id string) (dbmodel.InstanceArchivedDTO, error)
	GetOperationsStatsV2() ([]dbmodel.OperationStatEntryV2, error)
	ListDeletedInstanceIDs(amount int) ([]string, error)
	NumberOfOperationsForDeletedInstances() (int, error)
	TotalNumberOfInstancesArchived() (int, error)
	NumberOfDeletedInstances() (int, error)
	TotalNumberOfInstancesArchivedForGlobalAccountID(globalAccountID string, planID string) (int, error)
	GetAllOperations() ([]dbmodel.OperationDTO, error)
	ListInstancesArchived(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceArchivedDTO, int, int, error)
	GetBinding(instanceID string, bindingID string) (dbmodel.BindingDTO, dberr.Error)
	ListBindings(instanceID string) ([]dbmodel.BindingDTO, error)
	ListExpiredBindings() ([]dbmodel.BindingDTO, error)
	GetBindingsStatistics() (dbmodel.BindingStatsDTO, error)
	ListActions(instanceID string) ([]runtime.Action, error)
	GetTimeZone() (string, dberr.Error)
}

//go:generate mockery --name=WriteSession
type WriteSession interface {
	InsertInstance(instance dbmodel.InstanceDTO) dberr.Error
	UpdateInstance(instance dbmodel.InstanceDTO) dberr.Error
	DeleteInstance(instanceID string) dberr.Error
	InsertOperation(dto dbmodel.OperationDTO) dberr.Error
	UpdateOperation(dto dbmodel.OperationDTO) dberr.Error
	InsertEvent(level events.EventLevel, message, instanceID, operationID string) dberr.Error
	DeleteEvents(until time.Time) dberr.Error
	UpsertSubaccountState(state dbmodel.SubaccountStateDTO) dberr.Error
	DeleteState(id string) dberr.Error
	DeleteOperationByID(operationID string) dberr.Error
	InsertInstanceArchived(instance dbmodel.InstanceArchivedDTO) dberr.Error
	InsertBinding(binding dbmodel.BindingDTO) dberr.Error
	UpdateBinding(binding dbmodel.BindingDTO) dberr.Error
	DeleteBinding(instanceID, bindingID string) dberr.Error
	UpdateInstanceLastOperation(instanceID, operationID string) error
	InsertAction(actionType runtime.ActionType, instanceID, message, oldValue, newValue string) dberr.Error
}

type Transaction interface {
	Commit() dberr.Error
	RollbackUnlessCommitted()
}

//go:generate mockery --name=WriteSessionWithinTransaction
type WriteSessionWithinTransaction interface {
	WriteSession
	Transaction
}

type factory struct {
	connection *dbr.Connection
}

func NewFactory(connection *dbr.Connection) Factory {
	return &factory{
		connection: connection,
	}
}

func (sf *factory) NewReadSession() ReadSession {
	return readSession{
		session: sf.connection.NewSession(nil),
	}
}

func (sf *factory) NewWriteSession() WriteSession {
	return writeSession{
		session: sf.connection.NewSession(nil),
	}
}

func (sf *factory) NewSessionWithinTransaction() (WriteSessionWithinTransaction, dberr.Error) {
	dbSession := sf.connection.NewSession(nil)
	dbTransaction, err := dbSession.Begin()

	if err != nil {
		return nil, dberr.Internal("Failed to start transaction: %s", err)
	}

	return writeSession{
		session:     dbSession,
		transaction: dbTransaction,
	}, nil
}
