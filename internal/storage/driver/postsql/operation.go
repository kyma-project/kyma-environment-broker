package postsql

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/storage"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"k8s.io/apimachinery/pkg/util/wait"
)

type operations struct {
	postsql.Factory
	cipher Cipher
}

func NewOperation(sess postsql.Factory, cipher Cipher) *operations {
	return &operations{
		Factory: sess,
		cipher:  cipher,
	}
}

// InsertProvisioningOperation insert new ProvisioningOperation to storage
func (s *operations) InsertProvisioningOperation(operation internal.ProvisioningOperation) error {
	dto, err := s.provisioningOperationToDTO(&operation)
	if err != nil {
		return fmt.Errorf("while inserting provisioning operation (id: %s): %w", operation.ID, err)
	}
	return s.insert(dto)
}

// InsertOperation insert new Operation to storage
func (s *operations) InsertOperation(operation internal.Operation) error {
	dto, err := s.operationToDTO(&operation)

	if err != nil {
		return fmt.Errorf("while inserting operation (id: %s): %w", operation.ID, err)
	}

	return s.insert(dto)
}

// GetProvisioningOperationByID fetches the ProvisioningOperation by given ID, returns error if not found
func (s *operations) GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("while getting operation by ID: %w", err)
	}

	ret, err := s.toProvisioningOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// GetProvisioningOperationByInstanceID fetches the latest ProvisioningOperation by given instanceID, returns error if not found
func (s *operations) GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error) {

	operation, err := s.getByTypeAndInstanceID(instanceID, internal.OperationTypeProvision)
	if err != nil {
		return nil, err
	}
	ret, err := s.toProvisioningOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// UpdateOperation updates Operation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateOperation(op internal.Operation) (*internal.Operation, error) {
	op.UpdatedAt = time.Now()
	dto, err := s.operationToDTO(&op)

	if err != nil {
		return nil, fmt.Errorf("while converting Operation to DTO: %w", err)
	}

	lastErr := s.update(dto)
	op.Version = op.Version + 1

	return &op, lastErr
}

// UpdateProvisioningOperation updates ProvisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateProvisioningOperation(op internal.ProvisioningOperation) (*internal.ProvisioningOperation, error) {
	op.UpdatedAt = time.Now()
	dto, err := s.provisioningOperationToDTO(&op)

	if err != nil {
		return nil, fmt.Errorf("while converting Operation to DTO: %w", err)
	}

	lastErr := s.update(dto)
	op.Version = op.Version + 1

	return &op, lastErr
}

func (s *operations) ListProvisioningOperationsByInstanceID(instanceID string) ([]internal.ProvisioningOperation, error) {

	operations, err := s.listOperationsByInstanceIdAndType(instanceID, internal.OperationTypeProvision)
	if err != nil {
		return nil, fmt.Errorf("while loading operations list: %w", err)
	}

	ret, err := s.toProvisioningOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

func (s *operations) ListOperationsByInstanceID(instanceID string) ([]internal.Operation, error) {

	operations, err := s.listOperationsByInstanceId(instanceID)
	if err != nil {
		return nil, fmt.Errorf("while loading operations list: %w", err)
	}

	ret, err := s.toOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

func (s *operations) ListOperationsByInstanceIDGroupByType(instanceID string) (*internal.GroupedOperations, error) {

	operations, err := s.listOperationsByInstanceId(instanceID)
	if err != nil {
		return nil, fmt.Errorf("while loading operations list: %w", err)
	}

	grouped := internal.GroupedOperations{
		ProvisionOperations:      make([]internal.ProvisioningOperation, 0),
		DeprovisionOperations:    make([]internal.DeprovisioningOperation, 0),
		UpgradeClusterOperations: make([]internal.UpgradeClusterOperation, 0),
		UpdateOperations:         make([]internal.UpdatingOperation, 0),
	}

	for _, op := range operations {
		switch op.Type {
		case internal.OperationTypeProvision:
			ret, err := s.toProvisioningOperation(&op)
			if err != nil {
				return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
			}
			grouped.ProvisionOperations = append(grouped.ProvisionOperations, *ret)

		case internal.OperationTypeDeprovision:
			ret, err := s.toDeprovisioningOperation(&op)
			if err != nil {
				return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
			}
			grouped.DeprovisionOperations = append(grouped.DeprovisionOperations, *ret)

		case internal.OperationTypeUpgradeCluster:
			ret, err := s.toUpgradeClusterOperation(&op)
			if err != nil {
				return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
			}
			grouped.UpgradeClusterOperations = append(grouped.UpgradeClusterOperations, *ret)
		case internal.OperationTypeUpdate:
			ret, err := s.toUpdateOperation(&op)
			if err != nil {
				return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
			}
			grouped.UpdateOperations = append(grouped.UpdateOperations, *ret)
		case internal.OperationTypeUpgradeKyma:
			continue
		default:
			return nil, fmt.Errorf("while converting DTO to Operation: unrecognized type of operation")
		}
	}

	return &grouped, nil
}

// InsertDeprovisioningOperation insert new DeprovisioningOperation to storage
func (s *operations) InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error {

	dto, err := s.deprovisioningOperationToDTO(&operation)
	if err != nil {
		return fmt.Errorf("while converting Operation to DTO: %w", err)
	}

	return s.insert(dto)
}

// GetDeprovisioningOperationByID fetches the DeprovisioningOperation by given ID, returns error if not found
func (s *operations) GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("while getting operation by ID: %w", err)
	}

	ret, err := s.toDeprovisioningOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// GetDeprovisioningOperationByInstanceID fetches the latest DeprovisioningOperation by given instanceID, returns error if not found
func (s *operations) GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error) {
	operation, err := s.getByTypeAndInstanceID(instanceID, internal.OperationTypeDeprovision)
	if err != nil {
		return nil, err
	}
	ret, err := s.toDeprovisioningOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// UpdateDeprovisioningOperation updates DeprovisioningOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateDeprovisioningOperation(operation internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error) {
	operation.UpdatedAt = time.Now()

	dto, err := s.deprovisioningOperationToDTO(&operation)
	if err != nil {
		return nil, fmt.Errorf("while converting Operation to DTO: %w", err)
	}

	lastErr := s.update(dto)
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// ListDeprovisioningOperationsByInstanceID
func (s *operations) ListDeprovisioningOperationsByInstanceID(instanceID string) ([]internal.DeprovisioningOperation, error) {
	operations, err := s.listOperationsByInstanceIdAndType(instanceID, internal.OperationTypeDeprovision)
	if err != nil {
		return nil, err
	}

	ret, err := s.toDeprovisioningOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// ListDeprovisioningOperations lists deprovisioning operations
func (s *operations) ListDeprovisioningOperations() ([]internal.DeprovisioningOperation, error) {
	var lastErr dberr.Error

	operations, err := s.listOperationsByType(internal.OperationTypeDeprovision)
	if err != nil {
		return nil, lastErr
	}

	ret, err := s.toDeprovisioningOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// GetLastOperation returns Operation for given instance ID which is not in 'pending' state. Returns an error if the operation does not exist.
func (s *operations) GetLastOperation(instanceID string) (*internal.Operation, error) {
	session := s.Factory.NewReadSession()
	operation := dbmodel.OperationDTO{}
	op := internal.Operation{}
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {

		operation, lastErr = session.GetLastOperation(instanceID, []internal.OperationType{})
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with instance_id %s not exist", instanceID)
				return false, lastErr
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	err = json.Unmarshal([]byte(operation.Data), &op)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling operation data: %w", err)
	}
	op, err = s.toOperation(&operation, op)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// GetLastOperationByTypes returns Operation (with one of given types) for given instance ID which is not in 'pending' state. Returns an error if the operation does not exist.
func (s *operations) GetLastOperationByTypes(instanceID string, types []internal.OperationType) (*internal.Operation, error) {
	session := s.Factory.NewReadSession()
	dto, dbErr := session.GetLastOperation(instanceID, types)
	if dbErr != nil {
		return nil, dbErr
	}

	operation := internal.Operation{}
	err := json.Unmarshal([]byte(dto.Data), &operation)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling operation data: %w", err)
	}
	operation, err = s.toOperation(&dto, operation)
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

// GetOperationByID returns Operation with given ID. Returns an error if the operation does not exist.
func (s *operations) GetOperationByID(operationID string) (*internal.Operation, error) {
	op := internal.Operation{}
	dto, err := s.getByID(operationID)
	if err != nil {
		return nil, err
	}

	op, err = s.toOperation(dto, op)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(dto.Data), &op)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall operation data")
	}
	return &op, nil
}

func (s *operations) GetNotFinishedOperationsByType(operationType internal.OperationType) ([]internal.Operation, error) {
	session := s.Factory.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		dto, err := session.GetNotFinishedOperationsByType(operationType)
		if err != nil {
			return false, nil
		}
		operations = dto
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return s.toOperations(operations)
}

// TODO: If we use GROUP BY in the query, we can avoid this processing in the code
func (s *operations) GetOperationStatsByPlan() (map[string]internal.OperationStats, error) {
	entries, err := s.Factory.NewReadSession().GetOperationStats()
	if err != nil {
		return nil, err
	}
	result := make(map[string]internal.OperationStats)

	for _, entry := range entries {
		if !entry.PlanID.Valid || entry.PlanID.String == "" {
			continue
		}
		planId := entry.PlanID.String
		if _, ok := result[planId]; !ok {
			result[planId] = internal.OperationStats{
				Provisioning:   make(map[domain.LastOperationState]int),
				Deprovisioning: make(map[domain.LastOperationState]int),
			}
		}
		switch internal.OperationType(entry.Type) {
		case internal.OperationTypeProvision:
			result[planId].Provisioning[domain.LastOperationState(entry.State)] += 1
		case internal.OperationTypeDeprovision:
			result[planId].Deprovisioning[domain.LastOperationState(entry.State)] += 1
		}
	}

	return result, nil
}

func (s *operations) GetOperationStatsByPlanV2() ([]internal.OperationStatsV2, error) {
	entries, err := s.Factory.NewReadSession().GetOperationsStatsV2()
	if err != nil {
		return nil, err
	}
	var stats []internal.OperationStatsV2
	for _, entry := range entries {
		if !entry.PlanID.Valid || entry.PlanID.String == "" {
			continue
		}
		stats = append(stats, internal.OperationStatsV2{
			Count:  entry.Count,
			Type:   internal.OperationType(entry.Type),
			State:  domain.LastOperationState(entry.State),
			PlanID: entry.PlanID.String,
		})
	}
	return stats, nil
}

func (s *operations) GetOperationsForIDs(operationIDList []string) ([]internal.Operation, error) {
	session := s.Factory.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		dto, err := session.GetOperationsForIDs(operationIDList)
		if err != nil {
			return false, nil
		}
		operations = dto
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return s.toOperations(operations)
}

func (s *operations) ListOperations(filter dbmodel.OperationFilter) ([]internal.Operation, int, int, error) {
	session := s.Factory.NewReadSession()

	var (
		lastErr     error
		size, total int
		operations  = make([]dbmodel.OperationDTO, 0)
	)

	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operations, size, total, lastErr = session.ListOperations(filter)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, -1, -1, err
	}

	result, err := s.toOperations(operations)

	return result, size, total, err
}

func (s *operations) GetAllOperations() ([]internal.Operation, error) {
	session := s.Factory.NewReadSession()
	operations, err := session.GetAllOperations()
	if err != nil {
		return nil, err
	}
	result, err := s.toOperations(operations)
	return result, err
}

func (s *operations) ListOperationsInTimeRange(from, to time.Time) ([]internal.Operation, error) {
	session := s.Factory.NewReadSession()
	operations := make([]dbmodel.OperationDTO, 0)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		var err error
		operations, err = session.ListOperationsInTimeRange(from, to)
		if err != nil {
			if dberr.IsNotFound(err) {
				return true, nil
			}
			return false, fmt.Errorf("while listing the operations from the storage: %w", err)
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("while getting operations in range %v-%v: %w", from.Format(time.RFC822), to.Format(time.RFC822), err)
	}
	ret, err := s.toOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

func (s *operations) InsertUpdatingOperation(operation internal.UpdatingOperation) error {
	dto, err := s.updateOperationToDTO(&operation)
	if err != nil {
		return fmt.Errorf("while converting update operation (id: %s): %w", operation.Operation.ID, err)
	}

	return s.insert(dto)
}

func (s *operations) GetUpdatingOperationByID(operationID string) (*internal.UpdatingOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("while getting operation by ID: %w", err)
	}

	ret, err := s.toUpdateOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

func (s *operations) UpdateUpdatingOperation(operation internal.UpdatingOperation) (*internal.UpdatingOperation, error) {
	session := s.Factory.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := s.updateOperationToDTO(&operation)
	if err != nil {
		return nil, fmt.Errorf("while converting Operation to DTO: %w", err)
	}

	var lastErr error
	_ = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.Factory.NewReadSession().GetOperationByID(operation.Operation.ID)
			if lastErr != nil {
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.Operation.ID)
			return false, lastErr
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// ListUpdatingOperationsByInstanceID Lists all update operations for the given instance
func (s *operations) ListUpdatingOperationsByInstanceID(instanceID string) ([]internal.UpdatingOperation, error) {
	session := s.Factory.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceID, internal.OperationTypeUpdate)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toUpdateOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// InsertUpgradeClusterOperation insert new UpgradeClusterOperation to storage
func (s *operations) InsertUpgradeClusterOperation(operation internal.UpgradeClusterOperation) error {
	dto, err := s.upgradeClusterOperationToDTO(&operation)
	if err != nil {
		return fmt.Errorf("while converting upgrade cluser operation (id: %s): %w", operation.Operation.ID, err)
	}

	return s.insert(dto)
}

// UpdateUpgradeClusterOperation updates UpgradeClusterOperation, fails if not exists or optimistic locking failure occurs.
func (s *operations) UpdateUpgradeClusterOperation(operation internal.UpgradeClusterOperation) (*internal.UpgradeClusterOperation, error) {
	session := s.Factory.NewWriteSession()
	operation.UpdatedAt = time.Now()
	dto, err := s.upgradeClusterOperationToDTO(&operation)
	if err != nil {
		return nil, fmt.Errorf("while converting Operation to DTO: %w", err)
	}

	var lastErr error
	_ = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		lastErr = session.UpdateOperation(dto)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.Factory.NewReadSession().GetOperationByID(operation.Operation.ID)
			if lastErr != nil {
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.Operation.ID)
			return false, lastErr
		}
		return true, nil
	})
	operation.Version = operation.Version + 1
	return &operation, lastErr
}

// GetUpgradeClusterOperationByID fetches the UpgradeClusterOperation by given ID, returns error if not found
func (s *operations) GetUpgradeClusterOperationByID(operationID string) (*internal.UpgradeClusterOperation, error) {
	operation, err := s.getByID(operationID)
	if err != nil {
		return nil, fmt.Errorf("while getting operation by ID: %w", err)
	}
	ret, err := s.toUpgradeClusterOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

// ListUpgradeClusterOperationsByInstanceID Lists all upgrade cluster operations for the given instance
func (s *operations) ListUpgradeClusterOperationsByInstanceID(instanceID string) ([]internal.UpgradeClusterOperation, error) {
	session := s.Factory.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceID, internal.OperationTypeUpgradeCluster)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	ret, err := s.toUpgradeClusterOperationList(operations)
	if err != nil {
		return nil, fmt.Errorf("while converting DTO to Operation: %w", err)
	}

	return ret, nil
}

func (s *operations) DeleteByID(operationID string) error {
	return s.Factory.NewWriteSession().DeleteOperationByID(operationID)
}

func (s *operations) operationToDB(op internal.Operation) (dbmodel.OperationDTO, error) {
	err := s.cipher.EncryptSMCredentials(&op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while encrypting basic auth: %w", err)
	}
	err = s.cipher.EncryptKubeconfig(&op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while encrypting kubeconfig: %w", err)
	}
	pp, err := json.Marshal(op.ProvisioningParameters)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while marshal provisioning parameters: %w", err)
	}

	return dbmodel.OperationDTO{
		ID:                     op.ID,
		Type:                   op.Type,
		TargetOperationID:      op.ProvisionerOperationID,
		State:                  string(op.State),
		Description:            op.Description,
		UpdatedAt:              op.UpdatedAt,
		CreatedAt:              op.CreatedAt,
		Version:                op.Version,
		InstanceID:             op.InstanceID,
		ProvisioningParameters: storage.StringToSQLNullString(string(pp)),
		FinishedStages:         storage.StringToSQLNullString(strings.Join(op.FinishedStages, ",")),
	}, nil
}

func (s *operations) toOperation(dto *dbmodel.OperationDTO, existingOp internal.Operation) (internal.Operation, error) {
	provisioningParameters := internal.ProvisioningParameters{}
	if dto.ProvisioningParameters.Valid {
		err := json.Unmarshal([]byte(dto.ProvisioningParameters.String), &provisioningParameters)
		if err != nil {
			return internal.Operation{}, fmt.Errorf("while unmarshal provisioning parameters: %w", err)
		}
	}
	if provisioningParameters.ErsContext.SMOperatorCredentials != nil && strings.Contains(provisioningParameters.ErsContext.SMOperatorCredentials.ClientID, "!") {
		slog.Warn("decrypting credentials skipped because basic auth is in a plain text")
	} else {
		err := s.cipher.DecryptSMCredentialsUsingMode(&provisioningParameters)
		if err != nil {
			return internal.Operation{}, fmt.Errorf("while decrypting basic auth: %w", err)
		}
	}
	err := s.cipher.DecryptKubeconfigUsingMode(&provisioningParameters)
	if err != nil {
		slog.Warn("decrypting skipped because kubeconfig is in a plain text")
	}

	stages := make([]string, 0)
	finishedSteps := storage.SQLNullStringToString(dto.FinishedStages)
	for _, s := range strings.Split(finishedSteps, ",") {
		if s != "" {
			stages = append(stages, s)
		}
	}

	existingOp.ID = dto.ID
	existingOp.CreatedAt = dto.CreatedAt
	existingOp.UpdatedAt = dto.UpdatedAt
	existingOp.Type = dto.Type
	existingOp.ProvisionerOperationID = dto.TargetOperationID
	existingOp.State = domain.LastOperationState(dto.State)
	existingOp.InstanceID = dto.InstanceID
	existingOp.Description = dto.Description
	existingOp.Version = dto.Version
	existingOp.ProvisioningParameters = provisioningParameters
	existingOp.FinishedStages = stages

	return existingOp, nil
}

func (s *operations) toOperations(op []dbmodel.OperationDTO) ([]internal.Operation, error) {
	operations := make([]internal.Operation, 0)
	for _, o := range op {
		operation := internal.Operation{}
		operation, err := s.toOperation(&o, operation)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(o.Data), &operation)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshall operation data: %w", err)
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

func (s *operations) toProvisioningOperation(op *dbmodel.OperationDTO) (*internal.ProvisioningOperation, error) {
	if op.Type != internal.OperationTypeProvision {
		return nil, fmt.Errorf("expected operation type Provisioning, but was %s", op.Type)
	}
	var operation internal.ProvisioningOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall provisioning data: %w", err)
	}
	operation.Operation, err = s.toOperation(op, operation.Operation)
	if err != nil {
		return nil, err
	}
	return &operation, nil
}

func (s *operations) toProvisioningOperationList(ops []dbmodel.OperationDTO) ([]internal.ProvisioningOperation, error) {
	result := make([]internal.ProvisioningOperation, 0)

	for _, op := range ops {
		o, err := s.toProvisioningOperation(&op)
		if err != nil {
			return nil, fmt.Errorf("while converting to upgrade kyma operation: %w", err)
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) toDeprovisioningOperationList(ops []dbmodel.OperationDTO) ([]internal.DeprovisioningOperation, error) {
	result := make([]internal.DeprovisioningOperation, 0)

	for _, op := range ops {
		o, err := s.toDeprovisioningOperation(&op)
		if err != nil {
			return nil, fmt.Errorf("while converting to upgrade kyma operation: %w", err)
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) operationToDTO(op *internal.Operation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while serializing operation data %v: %w", op, err)
	}

	ret, err := s.operationToDB(*op)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while converting to operationDB %v: %w", op, err)
	}

	ret.Data = string(serialized)
	return ret, nil
}

func (s *operations) provisioningOperationToDTO(op *internal.ProvisioningOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while serializing provisioning data %v: %w", op, err)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while converting to operationDB %v: %w", op, err)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeProvision
	return ret, nil
}

func (s *operations) toDeprovisioningOperation(op *dbmodel.OperationDTO) (*internal.DeprovisioningOperation, error) {
	if op.Type != internal.OperationTypeDeprovision {
		return nil, fmt.Errorf("expected operation type Deprovision, but was %s", op.Type)
	}
	var operation internal.DeprovisioningOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall provisioning data: %w", err)
	}
	operation.Operation, err = s.toOperation(op, operation.Operation)
	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) deprovisioningOperationToDTO(op *internal.DeprovisioningOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while serializing deprovisioning data %v: %w", op, err)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while converting to operationDB %v: %w", op, err)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeDeprovision
	return ret, nil
}

func (s *operations) toOperationList(ops []dbmodel.OperationDTO) ([]internal.Operation, error) {
	result := make([]internal.Operation, 0)

	for _, op := range ops {

		var operation internal.Operation
		var err error
		err = json.Unmarshal([]byte(op.Data), &operation)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshall operation data: %w", err)
		}

		o, err := s.toOperation(&op, operation)
		if err != nil {
			return nil, fmt.Errorf("while converting to upgrade kyma operation: %w", err)
		}
		result = append(result, o)
	}

	return result, nil
}

func (s *operations) toUpgradeClusterOperation(op *dbmodel.OperationDTO) (*internal.UpgradeClusterOperation, error) {
	if op.Type != internal.OperationTypeUpgradeCluster {
		return nil, fmt.Errorf("expected operation type upgradeCluster, but was %s", op.Type)
	}
	var operation internal.UpgradeClusterOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall upgrade cluster data: %w", err)
	}
	operation.Operation, err = s.toOperation(op, operation.Operation)
	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) toUpgradeClusterOperationList(ops []dbmodel.OperationDTO) ([]internal.UpgradeClusterOperation, error) {
	result := make([]internal.UpgradeClusterOperation, 0)

	for _, op := range ops {
		o, err := s.toUpgradeClusterOperation(&op)
		if err != nil {
			return nil, fmt.Errorf("while converting to upgrade cluster operation: %w", err)
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) upgradeClusterOperationToDTO(op *internal.UpgradeClusterOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while serializing upgradeCluster data %v: %w", op, err)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while converting to operationDB %v: %w", op, err)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeUpgradeCluster
	return ret, nil
}

func (s *operations) updateOperationToDTO(op *internal.UpdatingOperation) (dbmodel.OperationDTO, error) {
	serialized, err := json.Marshal(op)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while serializing update data %v: %w", op, err)
	}

	ret, err := s.operationToDB(op.Operation)
	if err != nil {
		return dbmodel.OperationDTO{}, fmt.Errorf("while converting to operationDB %v: %w", op, err)
	}
	ret.Data = string(serialized)
	ret.Type = internal.OperationTypeUpdate
	return ret, nil
}

func (s *operations) toUpdateOperation(op *dbmodel.OperationDTO) (*internal.UpdatingOperation, error) {
	if op.Type != internal.OperationTypeUpdate {
		return nil, fmt.Errorf("expected operation type update, but was %s", op.Type)
	}
	var operation internal.UpdatingOperation
	var err error
	err = json.Unmarshal([]byte(op.Data), &operation)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall provisioning data")
	}
	operation.Operation, err = s.toOperation(op, operation.Operation)
	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) toUpdateOperationList(ops []dbmodel.OperationDTO) ([]internal.UpdatingOperation, error) {
	result := make([]internal.UpdatingOperation, 0)

	for _, op := range ops {
		o, err := s.toUpdateOperation(&op)
		if err != nil {
			return nil, fmt.Errorf("while converting to upgrade cluster operation: %w", err)
		}
		result = append(result, *o)
	}

	return result, nil
}

func (s *operations) getByID(id string) (*dbmodel.OperationDTO, error) {
	var lastErr dberr.Error
	session := s.Factory.NewReadSession()
	operation := dbmodel.OperationDTO{}

	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operation, lastErr = session.GetOperationByID(id)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("Operation with id %s not exist", id)
				return false, lastErr
			}
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return &operation, nil
}

func (s *operations) insert(dto dbmodel.OperationDTO) error {
	session := s.Factory.NewWriteSession()
	var lastErr error
	_ = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		lastErr = session.InsertOperation(dto)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	return lastErr
}

func (s *operations) getByInstanceID(id string) (*dbmodel.OperationDTO, error) {
	session := s.Factory.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operation, lastErr = session.GetOperationByInstanceID(id)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			return false, nil
		}
		return true, nil
	})

	return &operation, err
}

func (s *operations) getByTypeAndInstanceID(id string, opType internal.OperationType) (*dbmodel.OperationDTO, error) {
	session := s.Factory.NewReadSession()
	operation := dbmodel.OperationDTO{}
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operation, lastErr = session.GetOperationByTypeAndInstanceID(id, opType)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				lastErr = dberr.NotFound("operation does not exist")
				return false, lastErr
			}
			return false, nil
		}
		return true, nil
	})

	return &operation, err
}

func (s *operations) update(operation dbmodel.OperationDTO) error {
	session := s.Factory.NewWriteSession()

	var lastErr error
	_ = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		lastErr = session.UpdateOperation(operation)
		if lastErr != nil && dberr.IsNotFound(lastErr) {
			_, lastErr = s.Factory.NewReadSession().GetOperationByID(operation.ID)
			if dberr.IsNotFound(lastErr) {
				return false, lastErr
			}
			if lastErr != nil {
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("operation update conflict, operation ID: %s", operation.ID)
			return false, lastErr
		}
		return true, nil
	})
	return lastErr
}

func (s *operations) listOperationsByInstanceIdAndType(instanceId string, operationType internal.OperationType) ([]dbmodel.OperationDTO, error) {
	session := s.Factory.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error

	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operations, lastErr = session.GetOperationsByTypeAndInstanceID(instanceId, operationType)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return operations, lastErr
}

func (s *operations) listOperationsByType(operationType internal.OperationType) ([]dbmodel.OperationDTO, error) {
	session := s.Factory.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error

	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operations, lastErr = session.ListOperationsByType(operationType)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return operations, lastErr
}

func (s *operations) listOperationsByInstanceId(instanceId string) ([]dbmodel.OperationDTO, error) {
	session := s.Factory.NewReadSession()
	operations := []dbmodel.OperationDTO{}
	var lastErr dberr.Error

	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		operations, lastErr = session.GetOperationsByInstanceID(instanceId)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	return operations, lastErr
}
