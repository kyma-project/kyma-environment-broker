package postsql

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pivotal-cf/brokerapi/v12/domain"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/predicate"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Instance struct {
	postsql.Factory
	operations *operations
	cipher     Cipher
}

func (s *Instance) ListInstancesEncryptedUsingCFB(batchSize int) ([]internal.Instance, error) {
	var instances []dbmodel.InstanceDTO
	var err error
	err = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		var sessionError error
		instances, sessionError = s.Factory.NewReadSession().ListInstancesEncryptedUsingCFB(batchSize)
		if sessionError != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	var result []internal.Instance
	for _, dto := range instances {
		instance, err := s.toInstance(dto)
		if err != nil {
			return []internal.Instance{}, err
		}
		result = append(result, instance)
	}

	return result, nil
}

func (s *Instance) GetDistinctSubAccounts() ([]string, error) {
	sess := s.Factory.NewReadSession()
	var (
		subAccounts []string
		lastErr     dberr.Error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		subAccounts, lastErr = sess.GetDistinctSubAccounts()
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	return subAccounts, nil
}

func NewInstance(sess postsql.Factory, operations *operations, cipher Cipher) *Instance {
	return &Instance{
		Factory:    sess,
		operations: operations,
		cipher:     cipher,
	}
}

func (s *Instance) FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error) {
	sess := s.Factory.NewReadSession()
	var (
		instances []dbmodel.InstanceWithOperationDTO
		lastErr   dberr.Error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		instances, lastErr = sess.FindAllInstancesJoinedWithOperation(prct...)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	var result []internal.InstanceWithOperation
	for _, dto := range instances {
		inst, err := s.toInstance(dto.InstanceDTO)
		if err != nil {
			return nil, err
		}

		var isSuspensionOp bool

		switch internal.OperationType(dto.Type.String) {
		case internal.OperationTypeProvision:
			isSuspensionOp = false
		case internal.OperationTypeDeprovision:
			deprovOp, err := s.toDeprovisioningOp(&dto)
			if err != nil {
				slog.Error(fmt.Sprintf("while unmarshalling DTO deprovisioning operation data: %v", err))
			}
			isSuspensionOp = deprovOp.Temporary
		}

		result = append(result, internal.InstanceWithOperation{
			Instance:       inst,
			Type:           dto.Type,
			State:          dto.State,
			Description:    dto.Description,
			OpCreatedAt:    dto.OperationCreatedAt.Time,
			IsSuspensionOp: isSuspensionOp,
		})
	}

	return result, nil
}

func (s *Instance) toProvisioningOp(dto *dbmodel.InstanceWithOperationDTO) (*internal.ProvisioningOperation, error) {
	var provOp internal.ProvisioningOperation
	err := json.Unmarshal([]byte(dto.Data.String), &provOp)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall provisioning data")
	}

	return &provOp, nil
}

func (s *Instance) toDeprovisioningOp(dto *dbmodel.InstanceWithOperationDTO) (*internal.DeprovisioningOperation, error) {
	var deprovOp internal.DeprovisioningOperation
	err := json.Unmarshal([]byte(dto.Data.String), &deprovOp)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall deprovisioning data")
	}

	return &deprovOp, nil
}

func (s *Instance) FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error) {
	sess := s.Factory.NewReadSession()
	var instances []dbmodel.InstanceDTO
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		instances, lastErr = sess.FindAllInstancesForRuntimes(runtimeIdList)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instances with runtime IDs from list '%+q' not exist", runtimeIdList)
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	var result []internal.Instance
	for _, dto := range instances {
		inst, err := s.toInstance(dto)
		if err != nil {
			return []internal.Instance{}, err
		}
		result = append(result, inst)
	}

	return result, nil
}

func (s *Instance) FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error) {
	sess := s.Factory.NewReadSession()
	var (
		instances []dbmodel.InstanceDTO
		lastErr   dberr.Error
	)
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		instances, lastErr = sess.FindAllInstancesForSubAccounts(subAccountslist)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}

	var result []internal.Instance
	for _, dto := range instances {
		inst, err := s.toInstance(dto)
		if err != nil {
			return []internal.Instance{}, err
		}
		result = append(result, inst)
	}

	return result, nil
}

func (s *Instance) GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error) {
	sess := s.Factory.NewReadSession()
	var result int
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		count, err := sess.GetNumberOfInstancesForGlobalAccountID(globalAccountID)
		result = count
		return err == nil, nil
	})
	return result, err
}

// TODO: Wrap retries in single method WithRetries
func (s *Instance) GetByID(instanceID string) (*internal.Instance, error) {
	sess := s.Factory.NewReadSession()
	instanceDTO := dbmodel.InstanceDTO{}
	var lastErr dberr.Error
	err := wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		instanceDTO, lastErr = sess.GetInstanceByID(instanceID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instanceID)
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	instance, err := s.toInstance(instanceDTO)
	if err != nil {
		return nil, err
	}

	lastOp, err := s.operations.GetLastOperation(instanceID)
	if err != nil {
		if dberr.IsNotFound(err) {
			return &instance, nil
		}
		return nil, err
	}
	instance.InstanceDetails = lastOp.InstanceDetails
	return &instance, nil
}

func (s *Instance) toInstance(dto dbmodel.InstanceDTO) (internal.Instance, error) {
	var params internal.ProvisioningParameters
	err := json.Unmarshal([]byte(dto.ProvisioningParameters), &params)
	if err != nil {
		return internal.Instance{}, fmt.Errorf("while unmarshal parameters: %w", err)
	}

	err = s.cipher.DecryptSMCredentialsUsingMode(&params, dto.EncryptionMode)
	if err != nil {
		return internal.Instance{}, fmt.Errorf("while decrypting parameters: %w", err)
	}

	err = s.cipher.DecryptKubeconfigUsingMode(&params, dto.EncryptionMode)

	if err != nil {
		slog.Warn("decrypting skipped because kubeconfig is in a plain text")
	}

	return internal.Instance{
		InstanceID:                  dto.InstanceID,
		RuntimeID:                   dto.RuntimeID,
		GlobalAccountID:             dto.GlobalAccountID,
		SubscriptionGlobalAccountID: dto.SubscriptionGlobalAccountID,
		SubAccountID:                dto.SubAccountID,
		ServiceID:                   dto.ServiceID,
		ServiceName:                 dto.ServiceName,
		ServicePlanID:               dto.ServicePlanID,
		ServicePlanName:             dto.ServicePlanName,
		SubscriptionSecretName:      dto.SubscriptionSecretName,
		DashboardURL:                dto.DashboardURL,
		Parameters:                  params,
		ProviderRegion:              dto.ProviderRegion,
		CreatedAt:                   dto.CreatedAt,
		UpdatedAt:                   dto.UpdatedAt,
		DeletedAt:                   dto.DeletedAt,
		ExpiredAt:                   dto.ExpiredAt,
		Version:                     dto.Version,
		Provider:                    pkg.CloudProvider(dto.Provider),
	}, nil
}

func (s *Instance) toInstanceWithSubaccountState(dto dbmodel.InstanceWithSubaccountStateDTO) (internal.InstanceWithSubaccountState, error) {
	var params internal.ProvisioningParameters
	err := json.Unmarshal([]byte(dto.InstanceDTO.ProvisioningParameters), &params)
	if err != nil {
		return internal.InstanceWithSubaccountState{}, fmt.Errorf("while unmarshal parameters: %w", err)
	}

	err = s.cipher.DecryptSMCredentialsUsingMode(&params, dto.InstanceDTO.EncryptionMode)
	if err != nil {
		return internal.InstanceWithSubaccountState{}, fmt.Errorf("while decrypting parameters: %w", err)
	}

	err = s.cipher.DecryptKubeconfigUsingMode(&params, dto.InstanceDTO.EncryptionMode)
	if err != nil {
		slog.Warn("decrypting skipped because kubeconfig is in a plain text")
	}

	var betaEnabled, usedForProduction string
	if dto.BetaEnabled == nil {
		betaEnabled = ""
	} else {
		betaEnabled = *dto.BetaEnabled
	}
	if dto.UsedForProduction == nil {
		usedForProduction = ""
	} else {
		usedForProduction = *dto.UsedForProduction
	}
	return internal.InstanceWithSubaccountState{
		Instance: internal.Instance{InstanceID: dto.InstanceDTO.InstanceID,
			RuntimeID:                   dto.RuntimeID,
			GlobalAccountID:             dto.GlobalAccountID,
			SubscriptionGlobalAccountID: dto.SubscriptionGlobalAccountID,
			SubAccountID:                dto.SubAccountID,
			ServiceID:                   dto.ServiceID,
			ServiceName:                 dto.ServiceName,
			ServicePlanID:               dto.ServicePlanID,
			ServicePlanName:             dto.ServicePlanName,
			SubscriptionSecretName:      dto.SubscriptionSecretName,
			DashboardURL:                dto.DashboardURL,
			Parameters:                  params,
			ProviderRegion:              dto.ProviderRegion,
			CreatedAt:                   dto.InstanceDTO.CreatedAt,
			UpdatedAt:                   dto.InstanceDTO.UpdatedAt,
			DeletedAt:                   dto.DeletedAt,
			ExpiredAt:                   dto.ExpiredAt,
			Version:                     dto.InstanceDTO.Version,
			Provider:                    pkg.CloudProvider(dto.Provider)},
		BetaEnabled:       betaEnabled,
		UsedForProduction: usedForProduction,
	}, nil
}

func (s *Instance) Insert(instance internal.Instance) error {
	_, err := s.GetByID(instance.InstanceID)
	if err == nil {
		return dberr.AlreadyExists("instance with id %s already exist", instance.InstanceID)
	}

	dto, err := s.toInstanceDTO(instance)
	if err != nil {
		return err
	}

	sess := s.Factory.NewWriteSession()
	return wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		err := sess.InsertInstance(dto)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (s *Instance) Update(instance internal.Instance) (*internal.Instance, error) {
	sess := s.Factory.NewWriteSession()
	dto, err := s.toInstanceDTO(instance)
	if err != nil {
		return nil, err
	}
	var lastErr dberr.Error
	err = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		lastErr = sess.UpdateInstance(dto)

		switch {
		case dberr.IsNotFound(lastErr):
			_, lastErr = s.Factory.NewReadSession().GetInstanceByID(instance.InstanceID)
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instance.InstanceID)
			}
			if lastErr != nil {
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("instance update conflict, instance ID: %s", instance.InstanceID)
			return false, lastErr
		case lastErr != nil:
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	instance.Version = instance.Version + 1
	return &instance, nil
}

func (s *Instance) ReEncryptInstance(instance internal.Instance) error {
	sess := s.Factory.NewWriteSession()
	dto, err := s.toInstanceDTO(instance)
	if err != nil {
		return err
	}
	var lastErr dberr.Error
	err = wait.PollUntilContextTimeout(context.Background(), defaultRetryInterval, defaultRetryTimeout, true, func(ctx context.Context) (bool, error) {
		lastErr = sess.UpdateEncryptedDataInInstance(dto)

		switch {
		case dberr.IsNotFound(lastErr):
			_, lastErr = s.Factory.NewReadSession().GetInstanceByID(instance.InstanceID)
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("Instance with id %s not exist", instance.InstanceID)
			}
			if lastErr != nil {
				return false, nil
			}

			// the operation exists but the version is different
			lastErr = dberr.Conflict("instance update conflict, instance ID: %s", instance.InstanceID)
			return false, lastErr
		case lastErr != nil:
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return lastErr
	}
	instance.Version = instance.Version + 1
	return nil
}

func (s *Instance) toInstanceDTO(instance internal.Instance) (dbmodel.InstanceDTO, error) {
	err := s.cipher.EncryptSMCredentials(&instance.Parameters)
	if err != nil {
		return dbmodel.InstanceDTO{}, fmt.Errorf("while encrypting parameters: %w", err)
	}
	err = s.cipher.EncryptKubeconfig(&instance.Parameters)
	if err != nil {
		return dbmodel.InstanceDTO{}, fmt.Errorf("while encrypting kubeconfig: %w", err)
	}
	params, err := json.Marshal(instance.Parameters)
	if err != nil {
		return dbmodel.InstanceDTO{}, fmt.Errorf("while marshaling parameters: %w", err)
	}

	encryptionMode := s.cipher.GetEncryptionMode()

	return dbmodel.InstanceDTO{
		InstanceID:                  instance.InstanceID,
		RuntimeID:                   instance.RuntimeID,
		GlobalAccountID:             instance.GlobalAccountID,
		SubscriptionGlobalAccountID: instance.SubscriptionGlobalAccountID,
		SubAccountID:                instance.SubAccountID,
		ServiceID:                   instance.ServiceID,
		ServiceName:                 instance.ServiceName,
		ServicePlanID:               instance.ServicePlanID,
		ServicePlanName:             instance.ServicePlanName,
		SubscriptionSecretName:      instance.SubscriptionSecretName,
		DashboardURL:                instance.DashboardURL,
		ProvisioningParameters:      string(params),
		ProviderRegion:              instance.ProviderRegion,
		CreatedAt:                   instance.CreatedAt,
		UpdatedAt:                   instance.UpdatedAt,
		DeletedAt:                   instance.DeletedAt,
		ExpiredAt:                   instance.ExpiredAt,
		Version:                     instance.Version,
		Provider:                    string(instance.Provider),
		EncryptionMode:              encryptionMode,
	}, nil
}

func (s *Instance) Delete(instanceID string) error {
	sess := s.Factory.NewWriteSession()
	return sess.DeleteInstance(instanceID)
}

func (s *Instance) GetActiveInstanceStats() (internal.InstanceStats, error) {

	entries, err := s.Factory.NewReadSession().GetActiveInstanceStats()

	if err != nil {
		return internal.InstanceStats{}, err
	}

	result := internal.InstanceStats{
		PerGlobalAccountID: make(map[string]int),
		PerSubAcocuntID:    make(map[string]int),
	}
	for _, e := range entries {
		result.PerGlobalAccountID[e.GlobalAccountID] = e.Total
		result.TotalNumberOfInstances = result.TotalNumberOfInstances + e.Total
	}

	subEntries, err := s.Factory.NewReadSession().GetSubaccountsInstanceStats()

	if err != nil {
		return internal.InstanceStats{}, err
	}
	for _, e := range subEntries {
		result.PerSubAcocuntID[e.SubAccountID] = e.Total
	}
	return result, nil
}

func (s *Instance) GetERSContextStats() (internal.ERSContextStats, error) {

	entries, err := s.Factory.NewReadSession().GetERSContextStats()
	if err != nil {
		return internal.ERSContextStats{}, err
	}
	result := internal.ERSContextStats{
		LicenseType: make(map[string]int),
	}
	for _, e := range entries {
		result.LicenseType[strings.Trim(e.LicenseType.String, `"`)] += e.Total
	}
	return result, nil
}

func (s *Instance) List(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {

	dtos, count, totalCount, err := s.Factory.NewReadSession().ListInstances(filter)

	if err != nil {
		return []internal.Instance{}, 0, 0, err
	}
	var instances []internal.Instance
	for _, dto := range dtos {
		instance, err := s.toInstance(dto.InstanceDTO)
		if err != nil {
			return []internal.Instance{}, 0, 0, err
		}

		lastOp := internal.Operation{}
		err = json.Unmarshal([]byte(dto.OperationDTO.Data), &lastOp)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("while unmarshalling operation data: %w", err)
		}
		lastOp, err = s.operations.toOperation(&dto.OperationDTO, lastOp)
		if err != nil {
			return []internal.Instance{}, 0, 0, err
		}

		instance.InstanceDetails = lastOp.InstanceDetails
		instance.Reconcilable = instance.RuntimeID != "" && lastOp.Type != internal.OperationTypeDeprovision && lastOp.State != domain.InProgress
		instances = append(instances, instance)
	}
	return instances, count, totalCount, err
}

func (s *Instance) UpdateInstanceLastOperation(instanceID, operationID string) error {
	sess := s.Factory.NewWriteSession()
	return sess.UpdateInstanceLastOperation(instanceID, operationID)
}

func (s *Instance) ListWithSubaccountState(filter dbmodel.InstanceFilter) ([]internal.InstanceWithSubaccountState, int, int, error) {

	dtos, count, totalCount, err := s.Factory.NewReadSession().ListInstancesWithSubaccountStates(filter)

	if err != nil {
		return []internal.InstanceWithSubaccountState{}, 0, 0, err
	}
	var instances []internal.InstanceWithSubaccountState
	for _, dto := range dtos {
		instance, err := s.toInstanceWithSubaccountState(dto)
		if err != nil {
			return []internal.InstanceWithSubaccountState{}, 0, 0, err
		}

		lastOp := internal.Operation{}
		err = json.Unmarshal([]byte(dto.OperationDTO.Data), &lastOp)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("while unmarshalling operation data: %w", err)
		}
		lastOp, err = s.operations.toOperation(&dto.OperationDTO, lastOp)
		if err != nil {
			return []internal.InstanceWithSubaccountState{}, 0, 0, err
		}

		instance.InstanceDetails = lastOp.InstanceDetails
		instance.Reconcilable = instance.RuntimeID != "" && lastOp.Type != internal.OperationTypeDeprovision && lastOp.State != domain.InProgress
		instances = append(instances, instance)
	}
	return instances, count, totalCount, err
}

func (s *Instance) ListDeletedInstanceIDs(batchSize int) ([]string, error) {
	ids, err := s.Factory.NewReadSession().ListDeletedInstanceIDs(batchSize)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *Instance) DeletedInstancesStatistics() (internal.DeletedStats, error) {
	numberOfOperations, err := s.Factory.NewReadSession().NumberOfOperationsForDeletedInstances()
	if err != nil {
		return internal.DeletedStats{}, err
	}

	numberOfInstances, err := s.Factory.NewReadSession().NumberOfDeletedInstances()
	if err != nil {
		return internal.DeletedStats{}, err
	}
	return internal.DeletedStats{
		NumberOfDeletedInstances:              numberOfInstances,
		NumberOfOperationsForDeletedInstances: numberOfOperations,
	}, nil
}
