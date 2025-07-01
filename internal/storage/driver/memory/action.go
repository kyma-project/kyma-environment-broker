package memory

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/google/uuid"
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
		InstanceID: &instanceID,
		Message:    message,
		OldValue:   oldValue,
		NewValue:   newValue,
		CreatedAt:  time.Now(),
	})
	return nil
}

func (a *Action) UpdateAction(updated internal.Action) error {
	for i, action := range a.actions {
		if action.ID == updated.ID {
			a.actions[i] = updated
			return nil
		}
	}
	return dberr.NotFound("action with id %s does not exist", updated.ID)
}

func (a *Action) ListActionsByInstanceID(instanceID string) ([]internal.Action, error) {
	filtered := make([]internal.Action, 0)
	for _, action := range a.actions {
		if action.InstanceID != nil && *action.InstanceID == instanceID {
			filtered = append(filtered, action)
		}
	}
	return filtered, nil
}

func (a *Action) ListActionsByInstanceArchivedID(instanceArchivedID string) ([]internal.Action, error) {
	filtered := make([]internal.Action, 0)
	for _, action := range a.actions {
		if action.InstanceArchivedID != nil && *action.InstanceArchivedID == instanceArchivedID {
			filtered = append(filtered, action)
		}
	}
	return filtered, nil
}
