package postsql

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/events"
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/predicate"
	"golang.org/x/exp/slices"

	"github.com/gocraft/dbr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type readSession struct {
	session *dbr.Session
}

func (r readSession) GetTimeZone() (string, dberr.Error) {
	var timeZone string

	err := r.session.
		Select("current_setting('TIMEZONE')").
		LoadOne(&timeZone)

	if err != nil {
		if errors.Is(err, dbr.ErrNotFound) {
			return "", nil
		}
		return "", dberr.Internal("Failed to get time zone: %s", err)
	}

	return timeZone, nil
}

func (r readSession) GetBinding(instanceID string, bindingID string) (dbmodel.BindingDTO, dberr.Error) {
	var binding dbmodel.BindingDTO

	err := r.session.
		Select("*").
		From(BindingsTableName).
		Where(dbr.Eq("id", bindingID)).
		Where(dbr.Eq("instance_id", instanceID)).
		LoadOne(&binding)

	if err != nil {
		if errors.Is(err, dbr.ErrNotFound) {
			return dbmodel.BindingDTO{}, dberr.NotFound("Cannot find the Binding for bindingId:'%s'", bindingID)
		}
		return dbmodel.BindingDTO{}, dberr.Internal("Failed to get the Binding: %s", err)
	}

	return binding, nil
}

func (r readSession) ListBindings(instanceID string) ([]dbmodel.BindingDTO, error) {
	var bindings []dbmodel.BindingDTO
	if len(instanceID) == 0 {
		return bindings, fmt.Errorf("instanceID cannot be empty")
	}
	stmt := r.session.Select("*").From(BindingsTableName)
	if len(instanceID) != 0 {
		stmt.Where(dbr.Eq("instance_id", instanceID))
	}
	stmt.OrderBy("created_at")
	_, err := stmt.Load(&bindings)
	return bindings, err
}

func (r readSession) ListExpiredBindings() ([]dbmodel.BindingDTO, error) {
	currentTime := time.Now().UTC()
	var bindings []dbmodel.BindingDTO
	_, err := r.session.
		Select("id", "instance_id", "expires_at").
		From(BindingsTableName).
		Where(dbr.Lte("expires_at", currentTime)).
		Load(&bindings)

	if err != nil {
		return nil, fmt.Errorf("while getting expired bindings: %w", err)
	}

	return bindings, nil
}

func (r readSession) ListSubaccountStates() ([]dbmodel.SubaccountStateDTO, dberr.Error) {
	var states []dbmodel.SubaccountStateDTO

	_, err := r.session.
		Select("*").
		From(SubaccountStatesTableName).
		Load(&states)
	if err != nil {
		return nil, dberr.Internal("Failed to get subaccount states: %s", err)
	}
	return states, nil
}

func (r readSession) GetDistinctSubAccounts() ([]string, dberr.Error) {
	var subAccounts []string

	err := r.session.
		Select("distinct(sub_account_id)").
		From(InstancesTableName).
		Where(dbr.Neq("runtime_id", "")).
		LoadOne(&subAccounts)

	if err != nil {
		if errors.Is(err, dbr.ErrNotFound) {
			return []string{}, nil
		}
		return []string{}, dberr.Internal("Failed to get distinct subaccounts: %s", err)
	}

	return subAccounts, nil
}

// TODO: CAVEAT in case of large operations table
func (r readSession) getInstancesJoinedWithOperationStatement() *dbr.SelectStmt {
	join := fmt.Sprintf("%s.instance_id = %s.instance_id", InstancesTableName, OperationTableName)
	stmt := r.session.
		Select("instances.instance_id, instances.runtime_id, instances.global_account_id, instances.subscription_global_account_id, instances.service_id,"+
			" instances.service_plan_id, instances.dashboard_url, instances.provisioning_parameters, instances.created_at,"+
			" instances.updated_at, instances.deleted_at, instances.sub_account_id, instances.service_name, instances.service_plan_name,"+
			" instances.subscription_secret_name, instances.provider_region, instances.provider, operations.state, operations.description, operations.type, operations.created_at AS operation_created_at, operations.data").
		From(InstancesTableName).
		LeftJoin(OperationTableName, join)
	return stmt
}

func (r readSession) FindAllInstancesJoinedWithOperation(prct ...predicate.Predicate) ([]dbmodel.InstanceWithOperationDTO, dberr.Error) {
	var instances []dbmodel.InstanceWithOperationDTO

	stmt := r.getInstancesJoinedWithOperationStatement()
	for _, p := range prct {
		p.ApplyToPostgres(stmt)
	}

	if _, err := stmt.Load(&instances); err != nil {
		return nil, dberr.Internal("Failed to fetch all instances: %s", err)
	}

	return instances, nil
}

func (r readSession) GetInstanceByID(instanceID string) (dbmodel.InstanceDTO, dberr.Error) {
	var instance dbmodel.InstanceDTO

	err := r.session.
		Select("*").
		From(InstancesTableName).
		Where(dbr.Eq("instance_id", instanceID)).
		LoadOne(&instance)

	if err != nil {
		if errors.Is(err, dbr.ErrNotFound) {
			return dbmodel.InstanceDTO{}, dberr.NotFound("Cannot find Instance for instanceID:'%s'", instanceID)
		}
		return dbmodel.InstanceDTO{}, dberr.Internal("Failed to get Instance: %s", err)
	}

	return instance, nil
}

func (r readSession) GetInstanceArchivedByID(instanceId string) (dbmodel.InstanceArchivedDTO, error) {
	var result dbmodel.InstanceArchivedDTO

	err := r.session.Select("*").
		From(InstancesArchivedTableName).
		Where(dbr.Eq("instance_id", instanceId)).
		LoadOne(&result)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.InstanceArchivedDTO{}, dberr.NotFound("unable to find InstanceArchived %s", instanceId)
		}
		return dbmodel.InstanceArchivedDTO{}, dberr.Internal("Failed to get instance archived %s: %s", instanceId, err.Error())
	}
	return result, nil
}

func (r readSession) FindAllInstancesForRuntimes(runtimeIdList []string) ([]dbmodel.InstanceDTO, dberr.Error) {
	var instances []dbmodel.InstanceDTO

	err := r.session.
		Select("*").
		From(InstancesTableName).
		Where("runtime_id IN ?", runtimeIdList).
		LoadOne(&instances)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []dbmodel.InstanceDTO{}, dberr.NotFound("Cannot find Instances for runtime ID list: '%v'", runtimeIdList)
		}
		return []dbmodel.InstanceDTO{}, dberr.Internal("Failed to get Instances: %s", err)
	}
	return instances, nil
}

func (r readSession) FindAllInstancesForSubAccounts(subAccountslist []string) ([]dbmodel.InstanceDTO, dberr.Error) {
	var instances []dbmodel.InstanceDTO

	err := r.session.
		Select("*").
		From(InstancesTableName).
		Where("sub_account_id IN ?", subAccountslist).
		LoadOne(&instances)

	if err != nil {
		if err == dbr.ErrNotFound {
			return []dbmodel.InstanceDTO{}, nil
		}
		return []dbmodel.InstanceDTO{}, dberr.Internal("Failed to get Instances: %s", err)
	}
	return instances, nil
}

func (r readSession) GetLastOperation(instanceID string, types []internal.OperationType) (dbmodel.OperationDTO, dberr.Error) {
	inst := dbr.Eq("instance_id", instanceID)
	state := dbr.Neq("state", []string{internal.OperationStatePending, internal.OperationStateCanceled})
	condition := dbr.And(inst, state)
	if len(types) > 0 {
		condition = dbr.And(condition, dbr.Expr("type IN ?", types))
	}
	operation, err := r.getLastOperation(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OperationDTO{}, dberr.NotFound("for instance ID: %s %s", instanceID, err)
		default:
			return dbmodel.OperationDTO{}, err
		}
	}
	return operation, nil
}

func (r readSession) GetOperationByInstanceID(instanceId string) (dbmodel.OperationDTO, dberr.Error) {
	condition := dbr.Eq("instance_id", instanceId)
	operation, err := r.getOperation(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OperationDTO{}, dberr.NotFound("for instance_id: %s %s", instanceId, err)
		default:
			return dbmodel.OperationDTO{}, err
		}
	}
	return operation, nil
}

func (r readSession) GetOperationByID(opID string) (dbmodel.OperationDTO, dberr.Error) {
	condition := dbr.Eq("id", opID)
	operation, err := r.getOperation(condition)
	if err != nil {
		switch {
		case dberr.IsNotFound(err):
			return dbmodel.OperationDTO{}, dberr.NotFound("for ID: %s %s", opID, err)
		default:
			return dbmodel.OperationDTO{}, err
		}
	}
	return operation, nil
}

func (r readSession) ListOperations(filter dbmodel.OperationFilter) ([]dbmodel.OperationDTO, int, int, error) {
	var operations []dbmodel.OperationDTO

	stmt := r.session.Select("o.*").
		From(dbr.I(OperationTableName).As("o")).
		OrderBy("o.created_at")

	// Add pagination if provided
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	// Apply filtering if provided
	// TODO - assumes aliases o and i for operations and instances tables
	addOperationFilters(stmt, filter)

	_, err := stmt.Load(&operations)

	totalCount, err := r.getOperationCount(filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return operations,
		len(operations),
		totalCount,
		nil
}

func (r readSession) GetAllOperations() ([]dbmodel.OperationDTO, error) {
	var operations []dbmodel.OperationDTO

	_, err := r.session.Select("*").
		From(OperationTableName).
		Load(&operations)

	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) CountNotFinishedOperationsByInstanceID(instanceID string) (int, dberr.Error) {
	stateInProgress := dbr.Eq("state", domain.InProgress)
	statePending := dbr.Eq("state", internal.OperationStatePending)
	stateCondition := dbr.Or(statePending, stateInProgress)
	instanceIDCondition := dbr.Eq("instance_id", instanceID)

	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(OperationTableName).
		Where(stateCondition).
		Where(instanceIDCondition).
		LoadOne(&res)

	if err != nil {
		return 0, dberr.Internal("Failed to count operations: %s", err)
	}
	return res.Total, nil
}

func (r readSession) GetNotFinishedOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	stateInProgress := dbr.Eq("state", domain.InProgress)
	statePending := dbr.Eq("state", internal.OperationStatePending)
	stateCondition := dbr.Or(statePending, stateInProgress)
	typeCondition := dbr.Eq("type", operationType)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(stateCondition).
		Where(typeCondition).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationByTypeAndInstanceID(inID string, opType internal.OperationType) (dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	typeCondition := dbr.Eq("type", string(opType))
	var operation dbmodel.OperationDTO

	err := r.session.
		Select("*").
		From(OperationTableName).
		Where(idCondition).
		Where(typeCondition).
		OrderDesc(CreatedAtField).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return operation, nil
}

func (r readSession) GetOperationsByTypeAndInstanceID(inID string, opType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	typeCondition := dbr.Eq("type", string(opType))
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(idCondition).
		Where(typeCondition).
		OrderDesc(CreatedAtField).
		Load(&operations)

	if err != nil {
		return []dbmodel.OperationDTO{}, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationsByInstanceID(inID string) ([]dbmodel.OperationDTO, dberr.Error) {
	idCondition := dbr.Eq("instance_id", inID)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(idCondition).
		OrderDesc(CreatedAtField).
		Load(&operations)

	if err != nil {
		return []dbmodel.OperationDTO{}, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) GetOperationsForIDs(opIDlist []string) ([]dbmodel.OperationDTO, dberr.Error) {
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where("id IN ?", opIDlist).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

func (r readSession) ListOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, dberr.Error) {
	typeCondition := dbr.Eq("type", operationType)
	var operations []dbmodel.OperationDTO

	_, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(typeCondition).
		Load(&operations)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}
	return operations, nil
}

// TODO quite a tough query as for now
func (r readSession) ListOperationsInTimeRange(from, to time.Time) ([]dbmodel.OperationDTO, error) {
	var ops []dbmodel.OperationDTO
	condition := dbr.Or(
		dbr.And(dbr.Gte("created_at", from), dbr.Lte("created_at", to)),
		dbr.And(dbr.Gte("updated_at", from), dbr.Lte("updated_at", to)),
	)

	stmt := r.session.
		Select("*").
		From(OperationTableName).
		Where(condition)

	_, err := stmt.Load(&ops)
	if err != nil {
		return nil, dberr.Internal("Failed to get operations: %s", err)
	}

	return ops, nil
}

func (r readSession) getOperation(condition dbr.Builder) (dbmodel.OperationDTO, dberr.Error) {
	var operation dbmodel.OperationDTO

	err := r.session.
		Select("*").
		From(OperationTableName).
		Where(condition).
		LoadOne(&operation)

	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	return operation, nil
}

func (r readSession) getLastOperation(condition dbr.Builder) (dbmodel.OperationDTO, dberr.Error) {
	var operation dbmodel.OperationDTO

	count, err := r.session.
		Select("*").
		From(OperationTableName).
		Where(condition).
		OrderDesc(CreatedAtField).
		Limit(1).
		Load(&operation)
	if err != nil {
		if err == dbr.ErrNotFound {
			return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
		}
		return dbmodel.OperationDTO{}, dberr.Internal("Failed to get operation: %s", err)
	}
	if count == 0 {
		return dbmodel.OperationDTO{}, dberr.NotFound("cannot find operation: %s", err)
	}

	return operation, nil
}

// We fetch provision, deprovision operations only because in the callers stack we filter out other types
func (r readSession) GetOperationStats() ([]dbmodel.OperationStatEntry, error) {
	var rows []dbmodel.OperationStatEntry
	_, err := r.session.Select("type", "state", "provisioning_parameters ->> 'plan_id' AS plan_id").
		Where("type IN (?, ?)", "provision", "deprovision").
		From(OperationTableName).
		Load(&rows)
	return rows, err
}

func (r readSession) GetOperationsStatsV2() ([]dbmodel.OperationStatEntryV2, error) {
	var rows []dbmodel.OperationStatEntryV2

	_, err := r.session.Select("COUNT(*)", "type", "state", "provisioning_parameters ->> 'plan_id' AS plan_id").
		From(OperationTableName).
		Where("state = ?", "in progress").
		Where("type IN (?, ?, ?)", "provision", "deprovision", "update").
		GroupBy("type", "state", "provisioning_parameters ->> 'plan_id'").
		Load(&rows)
	return rows, err
}

func (r readSession) GetActiveInstanceStats() ([]dbmodel.InstanceByGlobalAccountIDStatEntry, error) {
	var rows []dbmodel.InstanceByGlobalAccountIDStatEntry
	var stmt *dbr.SelectStmt
	filter := dbmodel.InstanceFilter{
		States: []dbmodel.InstanceState{dbmodel.InstanceNotDeprovisioned},
	}

	// Find and join the last operation for each instance matching the state filter(s).
	stmt = r.session.
		Select(fmt.Sprintf("%s.global_account_id", InstancesTableName), "count(*) as total").
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.last_operation_id = o1.id", InstancesTableName)).
		Where("deleted_at = '0001-01-01T00:00:00.000Z'").
		Where(buildInstanceStateFilters("o1", filter)).
		GroupBy(fmt.Sprintf("%s.global_account_id", InstancesTableName))

	_, err := stmt.Load(&rows)
	return rows, err
}

func (r readSession) GetSubaccountsInstanceStats() ([]dbmodel.InstanceBySubAccountIDStatEntry, error) {
	var rows []dbmodel.InstanceBySubAccountIDStatEntry
	var stmt *dbr.SelectStmt
	filter := dbmodel.InstanceFilter{
		States: []dbmodel.InstanceState{dbmodel.InstanceNotDeprovisioned},
	}
	// Find and join the last operation for each instance matching the state filter(s).
	stmt = r.session.
		Select(fmt.Sprintf("%s.sub_account_id", InstancesTableName), "count(*) as total").
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.last_operation_id = o1.id", InstancesTableName)).
		Where("deleted_at = '0001-01-01T00:00:00.000Z'").
		Where(buildInstanceStateFilters("o1", filter)).
		GroupBy(fmt.Sprintf("%s.sub_account_id", InstancesTableName)).
		Having(fmt.Sprintf("count(*) > 1"))

	_, err := stmt.Load(&rows)
	return rows, err
}

func (r readSession) GetERSContextStats() ([]dbmodel.InstanceERSContextStatsEntry, error) {
	var rows []dbmodel.InstanceERSContextStatsEntry
	// group existing instances by license_Type from the last operation
	_, err := r.session.SelectBySql(`
SELECT count(*) as total, (o.provisioning_parameters -> 'ers_context' -> 'license_type')::VARCHAR AS license_type
FROM instances i
         INNER JOIN operations o ON i.last_operation_id = o.id
WHERE i.deleted_at = '0001-01-01T00:00:00.000Z'
GROUP BY license_type;`).Load(&rows)
	return rows, err
}

func (r readSession) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(InstancesTableName).
		Where(dbr.Eq("global_account_id", globalAccountID)).
		Where(dbr.Eq("deleted_at", "0001-01-01T00:00:00.000Z")).
		LoadOne(&res)

	return res.Total, err
}

func (r readSession) ListInstances(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceWithExtendedOperationDTO, int, int, error) {
	var instances []dbmodel.InstanceWithExtendedOperationDTO

	// select an instance with a last operation
	stmt := r.session.Select("o.data", "o.state", "o.type", fmt.Sprintf("%s.*", InstancesTableName)).
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o"), fmt.Sprintf("%s.last_operation_id = o.id", InstancesTableName)).
		OrderBy(fmt.Sprintf("%s.%s", InstancesTableName, CreatedAtField))

	if len(filter.States) > 0 || filter.Suspended != nil {
		stateFilters := buildInstanceStateFilters("o", filter)
		stmt.Where(stateFilters)
	}

	// Add pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt = stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	addInstanceFilters(stmt, filter, "o")

	_, err := stmt.Load(&instances)
	if err != nil {
		return nil, -1, -1, fmt.Errorf("while fetching instances: %w", err)
	}

	totalCount, err := r.getInstanceCountByLastOperationID(filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return instances,
		len(instances),
		totalCount,
		nil
}

func (r readSession) ListInstancesWithSubaccountStates(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceWithSubaccountStateDTO, int, int, error) {
	var instances []dbmodel.InstanceWithSubaccountStateDTO

	// Base select and order by created at
	var stmt *dbr.SelectStmt

	// select an instance with a last operation
	stmt = r.session.Select("o1.data", "o1.state", "o1.type", fmt.Sprintf("%s.*", InstancesTableName), "ss.beta_enabled", "ss.used_for_production").
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.last_operation_id = o1.id", InstancesTableName)).
		LeftJoin(dbr.I(SubaccountStatesTableName).As("ss"), fmt.Sprintf("%s.sub_account_id = ss.id", InstancesTableName)).
		OrderBy(fmt.Sprintf("%s.%s", InstancesTableName, CreatedAtField))

	if len(filter.States) > 0 || filter.Suspended != nil {
		stateFilters := buildInstanceStateFilters("o1", filter)
		stmt.Where(stateFilters)
	}

	// Add pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		stmt = stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	addInstanceFilters(stmt, filter, "o1")

	_, err := stmt.Load(&instances)
	if err != nil {
		return nil, -1, -1, fmt.Errorf("while fetching instances: %w", err)
	}

	// getInstanceCount is appropriate for this query because we added only left join without any additional selection/filtering
	totalCount, err := r.getInstanceCountByLastOperationID(filter)
	if err != nil {
		return nil, -1, -1, err
	}

	return instances,
		len(instances),
		totalCount,
		nil
}

func (r readSession) ListEvents(filter events.EventFilter) ([]events.EventDTO, error) {
	var events []events.EventDTO
	stmt := r.session.Select("*").From("events")
	if len(filter.InstanceIDs) != 0 {
		stmt.Where(dbr.Eq("instance_id", filter.InstanceIDs))
	}
	if len(filter.OperationIDs) != 0 {
		stmt.Where(dbr.Eq("operation_id", filter.OperationIDs))
	}
	stmt.OrderBy("created_at")
	_, err := stmt.Load(&events)
	return events, err
}

func (r readSession) getInstanceCountByLastOperationID(filter dbmodel.InstanceFilter) (int, error) {
	var res struct {
		Total int
	}

	var stmt *dbr.SelectStmt
	stmt = r.session.
		Select("count(*) as total").
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.last_operation_id = o1.id", InstancesTableName))

	if len(filter.States) > 0 || filter.Suspended != nil {
		stateFilters := buildInstanceStateFilters("o1", filter)
		stmt.Where(stateFilters)
	}

	addInstanceFilters(stmt, filter, "o1")
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func (r readSession) getInstanceCount(filter dbmodel.InstanceFilter) (int, error) {
	var res struct {
		Total int
	}
	var stmt *dbr.SelectStmt

	stmt = r.session.
		Select("count(*) as total").
		From(InstancesTableName).
		Join(dbr.I(OperationTableName).As("o1"), fmt.Sprintf("%s.instance_id = o1.instance_id", InstancesTableName)).
		LeftJoin(dbr.I(OperationTableName).As("o2"), fmt.Sprintf("%s.instance_id = o2.instance_id AND o1.created_at < o2.created_at AND o2.state NOT IN ('%s', '%s')", InstancesTableName, internal.OperationStatePending, internal.OperationStateCanceled)).
		Where("o2.created_at IS NULL").
		Where(fmt.Sprintf("o1.state NOT IN ('%s', '%s')", internal.OperationStatePending, internal.OperationStateCanceled))

	if len(filter.States) > 0 || filter.Suspended != nil {
		stateFilters := buildInstanceStateFilters("o1", filter)
		stmt.Where(stateFilters)
	}

	addInstanceFilters(stmt, filter, "o1")
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func buildInstanceStateFilters(table string, filter dbmodel.InstanceFilter) dbr.Builder {
	var exprs []dbr.Builder
	for _, s := range filter.States {
		switch s {
		case dbmodel.InstanceSucceeded:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Succeeded),
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
			))
		case dbmodel.InstanceFailed:
			exprs = append(exprs, dbr.And(
				dbr.Or(
					dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeProvision),
					dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Failed),
			))
		case dbmodel.InstanceError:
			exprs = append(exprs, dbr.And(
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeProvision),
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Failed),
			))
		case dbmodel.InstanceProvisioning:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeProvision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceDeprovisioning:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceUpgrading:
			exprs = append(exprs, dbr.And(
				dbr.Like(fmt.Sprintf("%s.type", table), "upgrade%"),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceUpdating:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeUpdate),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.InProgress),
			))
		case dbmodel.InstanceDeprovisioned:
			exprs = append(exprs, dbr.And(
				dbr.Eq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Eq(fmt.Sprintf("%s.state", table), domain.Succeeded),
			))
		case dbmodel.InstanceNotDeprovisioned:
			exprs = append(exprs, dbr.Or(
				dbr.Neq(fmt.Sprintf("%s.type", table), internal.OperationTypeDeprovision),
				dbr.Neq(fmt.Sprintf("%s.state", table), domain.Succeeded),
			))
		}
	}
	if filter.Suspended != nil && *filter.Suspended {
		exprs = append(exprs, dbr.Expr("((instances.provisioning_parameters::JSONB->>'ers_context')::JSONB->>'active')::BOOLEAN IS false"))
	}

	return dbr.Or(exprs...)
}

func addInstanceFilters(stmt *dbr.SelectStmt, filter dbmodel.InstanceFilter, table string) {
	if len(filter.GlobalAccountIDs) > 0 {
		stmt.Where("instances.global_account_id IN ?", filter.GlobalAccountIDs)
	}
	if len(filter.SubAccountIDs) > 0 {
		stmt.Where("instances.sub_account_id IN ?", filter.SubAccountIDs)
	}
	if len(filter.InstanceIDs) > 0 {
		stmt.Where("instances.instance_id IN ?", filter.InstanceIDs)
	}
	if len(filter.RuntimeIDs) > 0 {
		stmt.Where("instances.runtime_id IN ?", filter.RuntimeIDs)
	}
	if len(filter.Regions) > 0 {
		stmt.Where("instances.provider_region IN ?", filter.Regions)
	}
	if len(filter.Plans) > 0 {
		stmt.Where("instances.service_plan_name IN ?", filter.Plans)
	}
	if len(filter.PlanIDs) > 0 {
		stmt.Where("instances.service_plan_id IN ?", filter.PlanIDs)
	}
	if len(filter.Shoots) > 0 {
		shootNameMatch := fmt.Sprintf(`^(%s)$`, strings.Join(filter.Shoots, "|"))
		stmt.Where(fmt.Sprintf("%s.data::json->>'shoot_name' ~ ?", table), shootNameMatch)
	}

	if filter.Expired != nil {
		if *filter.Expired {
			stmt.Where("instances.expired_at IS NOT NULL")
		}
		if !*filter.Expired {
			stmt.Where("instances.expired_at IS NULL")
		}
	}

	if filter.DeletionAttempted != nil {
		if *filter.DeletionAttempted {
			stmt.Where("instances.deleted_at != '0001-01-01T00:00:00.000Z'")
		}
		if !*filter.DeletionAttempted {
			stmt.Where("instances.deleted_at = '0001-01-01T00:00:00.000Z'")
		}
	}

	if filter.BindingExists != nil && *filter.BindingExists {
		stmt.Where("exists (select instance_id from bindings where bindings.instance_id=instances.instance_id)")
	}
}

func addOperationFilters(stmt *dbr.SelectStmt, filter dbmodel.OperationFilter) {
	if len(filter.States) > 0 {
		stmt.Where("o.state IN ?", filter.States)
	}
	if filter.InstanceFilter != nil {
		fi := filter.InstanceFilter
		if slices.Contains(filter.States, string(dbmodel.InstanceDeprovisioned)) {
			stmt.LeftJoin(dbr.I(InstancesTableName).As("i"), "i.instance_id = o.instance_id").
				Where("i.instance_id IS NULL")
		}
		if len(fi.InstanceIDs) != 0 {
			stmt.Where("o.instance_id IN ?", fi.InstanceIDs)
		}
	}
}

func (r readSession) getOperationCount(filter dbmodel.OperationFilter) (int, error) {
	var res struct {
		Total int
	}
	stmt := r.session.Select("count(1) as total").
		From(dbr.I(OperationTableName).As("o"))

	// TODO - assumes aliases o and i for operations and instances tables
	addOperationFilters(stmt, filter)
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func (r readSession) ListDeletedInstanceIDs(amount int) ([]string, error) {
	rows, err := r.session.Query(fmt.Sprintf("select distinct(instance_id) from operations where instance_id not in (select instance_id from instances) limit %d", amount))
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, err
}

func (r readSession) NumberOfOperationsForDeletedInstances() (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(OperationTableName).
		Where("instance_id not in (select instance_id from instances)").
		LoadOne(&res)
	return res.Total, err
}

func (r readSession) NumberOfDeletedInstances() (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(distinct(instance_id)) as total").
		From(OperationTableName).
		Where("instance_id not in (select instance_id from instances)").
		LoadOne(&res)
	return res.Total, err
}

func (r readSession) TotalNumberOfInstancesArchived() (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(InstancesArchivedTableName).
		LoadOne(&res)
	return res.Total, err
}

func (r readSession) TotalNumberOfInstancesArchivedForGlobalAccountID(globalAccountID string, planID string) (int, error) {
	var res struct {
		Total int
	}
	err := r.session.Select("count(*) as total").
		From(InstancesArchivedTableName).
		Where(dbr.Eq("global_account_id", globalAccountID)).
		Where(dbr.Eq("plan_id", planID)).
		Where(dbr.Eq("provisioning_state", domain.Succeeded)).
		LoadOne(&res)

	return res.Total, err
}

func (r readSession) ListInstancesArchived(filter dbmodel.InstanceFilter) ([]dbmodel.InstanceArchivedDTO, int, int, error) {
	var instancesArchived []dbmodel.InstanceArchivedDTO

	stmt := r.session.Select("*").
		From(InstancesArchivedTableName).
		OrderDesc("last_deprovisioning_finished_at")

	if filter.Page > 0 && filter.PageSize > 0 {
		stmt.Paginate(uint64(filter.Page), uint64(filter.PageSize))
	}

	addInstanceArchivedFilter(stmt, filter)

	_, err := stmt.Load(&instancesArchived)

	totalCount, err := r.getInstanceArchivedCount(filter)
	if err != nil {
		return []dbmodel.InstanceArchivedDTO{}, -1, -1, err
	}

	return instancesArchived, len(instancesArchived), totalCount, nil
}

func (r readSession) GetBindingsStatistics() (dbmodel.BindingStatsDTO, error) {
	dto := dbmodel.BindingStatsDTO{}
	statement := r.session.Select("max(extract(epoch from AGE(now(), expires_at))) as seconds_since_earliest_expiration").From(BindingsTableName)

	err := statement.LoadOne(&dto)
	if err != nil {
		return dbmodel.BindingStatsDTO{}, err
	}
	if dto.SecondsSinceEarliestExpiration == nil {
		dto.SecondsSinceEarliestExpiration = new(float64)
	}
	return dto, nil
}

func (r readSession) getInstanceArchivedCount(filter dbmodel.InstanceFilter) (int, error) {
	var res struct {
		Total int
	}
	stmt := r.session.Select("count(*) as total").
		From(InstancesArchivedTableName)

	addInstanceArchivedFilter(stmt, filter)
	err := stmt.LoadOne(&res)

	return res.Total, err
}

func (r readSession) ListActions(instanceID string) ([]runtime.Action, error) {
	var actions []runtime.Action
	stmt := r.session.Select("*").From(ActionsTableName)
	stmt.Where(dbr.Eq("instance_id", instanceID))
	stmt.OrderDesc("created_at")
	_, err := stmt.Load(&actions)
	return actions, err
}

func addInstanceArchivedFilter(stmt *dbr.SelectStmt, filter dbmodel.InstanceFilter) {
	if len(filter.InstanceIDs) > 0 {
		stmt.Where("instance_id IN ?", filter.InstanceIDs)
	}
	if len(filter.GlobalAccountIDs) > 0 {
		stmt.Where("global_account_id IN ?", filter.GlobalAccountIDs)
	}
	if len(filter.SubAccountIDs) > 0 {
		stmt.Where("subaccount_id IN ?", filter.SubAccountIDs)
	}
	if len(filter.Plans) > 0 {
		stmt.Where("plan_name IN ?", filter.Plans)
	}
	if len(filter.Regions) > 0 {
		stmt.Where("region IN ?", filter.Regions)
	}
	if len(filter.RuntimeIDs) > 0 {
		stmt.Where("last_runtime_id IN ?", filter.RuntimeIDs)
	}
	if len(filter.Shoots) > 0 {
		stmt.Where("shoot_name IN ?", filter.Shoots)
	}
}
