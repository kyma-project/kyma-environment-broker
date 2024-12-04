package postsql

import (
	"encoding/json"
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"k8s.io/apimachinery/pkg/util/wait"
)

type runtimeState struct {
	postsql.Factory

	cipher Cipher
}

func NewRuntimeStates(sess postsql.Factory, cipher Cipher) *runtimeState {
	return &runtimeState{
		Factory: sess,
		cipher:  cipher,
	}
}

func (s *runtimeState) DeleteByOperationID(operationID string) error {
	return s.NewWriteSession().DeleteRuntimeStatesByOperationID(operationID)
}

func (s *runtimeState) Insert(runtimeState internal.RuntimeState) error {
	state, err := s.runtimeStateToDB(runtimeState)
	if err != nil {
		return err
	}
	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.InsertRuntimeState(state)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (s *runtimeState) ListByRuntimeID(runtimeID string) ([]internal.RuntimeState, error) {
	sess := s.NewReadSession()
	states := make([]dbmodel.RuntimeStateDTO, 0)
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		states, lastErr = sess.ListRuntimeStateByRuntimeID(runtimeID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeStates not found")
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	result, err := s.toRuntimeStates(states)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *runtimeState) GetByOperationID(operationID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	state := dbmodel.RuntimeStateDTO{}
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		state, lastErr = sess.GetRuntimeStateByOperationID(operationID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for operation %s not found", operationID)
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}
	result, err := s.toRuntimeState(&state)
	if err != nil {
		return internal.RuntimeState{}, fmt.Errorf("while converting runtime states: %w", err)
	}

	return result, nil
}

func (s *runtimeState) GetLatestByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	var state dbmodel.RuntimeStateDTO
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		state, lastErr = sess.GetLatestRuntimeStateByRuntimeID(runtimeID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for runtime %s not found", runtimeID)
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}
	result, err := s.toRuntimeState(&state)
	if err != nil {
		return internal.RuntimeState{}, fmt.Errorf("while converting runtime state: %w", err)
	}

	return result, nil
}

func (s *runtimeState) GetLatestWithOIDCConfigByRuntimeID(runtimeID string) (internal.RuntimeState, error) {
	sess := s.NewReadSession()
	var state dbmodel.RuntimeStateDTO
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		state, lastErr = sess.GetLatestRuntimeStateWithOIDCConfigByRuntimeID(runtimeID)
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("RuntimeState for runtime %s not found", runtimeID)
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return internal.RuntimeState{}, lastErr
	}

	result, err := s.toRuntimeState(&state)
	if err != nil {
		return internal.RuntimeState{}, fmt.Errorf("while converting runtime state: %w", err)
	}
	if result.ClusterConfig.OidcConfig != nil {
		return result, nil
	}

	return internal.RuntimeState{}, fmt.Errorf("failed to find RuntimeState with OIDC config for runtime %s ", runtimeID)
}

func (s *runtimeState) runtimeStateToDB(state internal.RuntimeState) (dbmodel.RuntimeStateDTO, error) {
	kymaCfg, err := json.Marshal(state.KymaConfig)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, fmt.Errorf("while encoding kyma config: %w", err)
	}
	clusterCfg, err := json.Marshal(state.ClusterConfig)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, fmt.Errorf("while encoding cluster config: %w", err)
	}

	encKymaCfg, err := s.cipher.Encrypt(kymaCfg)
	if err != nil {
		return dbmodel.RuntimeStateDTO{}, fmt.Errorf("while encrypting kyma config: %w", err)
	}

	return dbmodel.RuntimeStateDTO{
		ID:            state.ID,
		CreatedAt:     state.CreatedAt,
		RuntimeID:     state.RuntimeID,
		OperationID:   state.OperationID,
		KymaConfig:    string(encKymaCfg),
		ClusterConfig: string(clusterCfg),
		K8SVersion:    state.ClusterConfig.KubernetesVersion,
	}, nil
}

func (s *runtimeState) toRuntimeState(dto *dbmodel.RuntimeStateDTO) (internal.RuntimeState, error) {
	var (
		kymaCfg    gqlschema.KymaConfigInput
		clusterCfg gqlschema.GardenerConfigInput
	)
	if dto.KymaConfig != "" {
		cfg, err := s.cipher.Decrypt([]byte(dto.KymaConfig))
		if err != nil {
			return internal.RuntimeState{}, fmt.Errorf("while decrypting kyma config: %w", err)
		}
		if err := json.Unmarshal(cfg, &kymaCfg); err != nil {
			return internal.RuntimeState{}, fmt.Errorf("while unmarshall kyma config: %w", err)
		}
	}
	if dto.ClusterConfig != "" {
		if err := json.Unmarshal([]byte(dto.ClusterConfig), &clusterCfg); err != nil {
			return internal.RuntimeState{}, fmt.Errorf("while unmarshall cluster config: %w", err)
		}
	}
	return internal.RuntimeState{
		ID:            dto.ID,
		CreatedAt:     dto.CreatedAt,
		RuntimeID:     dto.RuntimeID,
		OperationID:   dto.OperationID,
		KymaConfig:    kymaCfg,
		ClusterConfig: clusterCfg,
	}, nil
}

func (s *runtimeState) toRuntimeStates(states []dbmodel.RuntimeStateDTO) ([]internal.RuntimeState, error) {
	result := make([]internal.RuntimeState, 0)

	for _, state := range states {
		r, err := s.toRuntimeState(&state)
		if err != nil {
			return nil, fmt.Errorf("while converting runtime states: %w", err)
		}
		result = append(result, r)
	}

	return result, nil
}
