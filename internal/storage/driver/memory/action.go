package memory

import (
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type Action struct {
	actions []internal.Action
}

func NewAction() *Action {
	return &Action{
		actions: make([]internal.Action, 0),
	}
}

func (a *Action) InsertAction(actionType internal.ActionType, instanceID, message, oldValue, newValue string) error {
	a.actions = append(a.actions, internal.Action{
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

func (a *Action) ListActionsByInstanceID(instanceID string) ([]internal.Action, error) {
	filtered := make([]internal.Action, 0)
	for _, action := range a.actions {
		if action.InstanceID == instanceID {
			filtered = append(filtered, action)
		}
	}
	return filtered, nil
}
