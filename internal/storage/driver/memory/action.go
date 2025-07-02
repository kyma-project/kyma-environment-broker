package memory

import (
	"sort"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"

	"github.com/google/uuid"
)

type Action struct {
	actions []runtime.Action
}

func NewAction() *Action {
	return &Action{
		actions: make([]runtime.Action, 0),
	}
}

func (a *Action) InsertAction(actionType runtime.ActionType, instanceID, message, oldValue, newValue string) error {
	a.actions = append(a.actions, runtime.Action{
		ID:         uuid.NewString(),
		Type:       actionType,
		InstanceID: instanceID,
		Message:    message,
		OldValue:   oldValue,
		NewValue:   newValue,
		CreatedAt:  time.Now(),
	})
	return nil
}

func (a *Action) ListActionsByInstanceID(instanceID string) ([]runtime.Action, error) {
	filtered := make([]runtime.Action, 0)
	for _, action := range a.actions {
		if action.InstanceID == instanceID {
			filtered = append(filtered, action)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})
	return filtered, nil
}
