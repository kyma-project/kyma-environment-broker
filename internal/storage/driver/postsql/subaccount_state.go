package postsql

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"k8s.io/apimachinery/pkg/util/wait"
)

type SubaccountState struct {
	postsql.Factory
}

func NewSubaccountStates(sess postsql.Factory) *SubaccountState {
	return &SubaccountState{
		Factory: sess,
	}
}

func (s *SubaccountState) UpsertState(subaccountState internal.SubaccountState) error {
	state, err := s.subaccountStateToDB(subaccountState)
	if err != nil {
		return err
	}
	sess := s.NewWriteSession()
	return wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		err := sess.UpsertSubaccountState(state)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (s *SubaccountState) DeleteState(subaccountID string) error {
	sess := s.NewWriteSession()
	return sess.DeleteState(subaccountID)
}

func (s *SubaccountState) ListStates() ([]internal.SubaccountState, error) {
	sess := s.NewReadSession()
	states := make([]dbmodel.SubaccountStateDTO, 0)
	var lastErr dberr.Error
	err := wait.PollImmediate(defaultRetryInterval, defaultRetryTimeout, func() (bool, error) {
		states, lastErr = sess.ListSubaccountStates()
		if lastErr != nil {
			if dberr.IsNotFound(lastErr) {
				return false, dberr.NotFound("subaccount_states not found")
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, lastErr
	}
	result, err := s.toSubaccountStates(states)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *SubaccountState) subaccountStateToDB(state internal.SubaccountState) (dbmodel.SubaccountStateDTO, error) {
	return dbmodel.SubaccountStateDTO{
		ID:                state.ID,
		BetaEnabled:       state.BetaEnabled,
		UsedForProduction: state.UsedForProduction,
		ModifiedAt:        state.ModifiedAt,
	}, nil
}

func (s *SubaccountState) toSubaccountState(dto *dbmodel.SubaccountStateDTO) (internal.SubaccountState, error) {
	return internal.SubaccountState{
		ID:                dto.ID,
		BetaEnabled:       dto.BetaEnabled,
		UsedForProduction: dto.UsedForProduction,
		ModifiedAt:        dto.ModifiedAt,
	}, nil
}

func (s *SubaccountState) toSubaccountStates(states []dbmodel.SubaccountStateDTO) ([]internal.SubaccountState, error) {
	result := make([]internal.SubaccountState, 0)
	for _, state := range states {
		r, err := s.toSubaccountState(&state)
		if err != nil {
			return nil, fmt.Errorf("while converting subaccount states: %v", err)
		}
		result = append(result, r)
	}
	return result, nil
}
