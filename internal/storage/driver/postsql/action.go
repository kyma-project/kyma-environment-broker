package postsql

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
)

type Action struct {
	postsql.Factory
}

func NewAction(sess postsql.Factory) *Action {
	return &Action{
		Factory: sess,
	}
}

func (a *Action) InsertAction(actionType internal.ActionType, instanceID, message, oldValue, newValue string) error {
	return a.Factory.NewWriteSession().InsertAction(actionType, instanceID, message, oldValue, newValue)
}

func (a *Action) ListActionsByInstanceID(instanceID string) ([]internal.Action, error) {
	return a.Factory.NewReadSession().ListActions(instanceID)
}
