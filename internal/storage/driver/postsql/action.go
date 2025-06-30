package postsql

import (
	"fmt"
	"log/slog"

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

func (a *Action) InsertAction(actionType internal.ActionType, instanceID, message, oldValue, newValue string) {
	sess := a.Factory.NewWriteSession()
	if err := sess.InsertAction(actionType, instanceID, message, oldValue, newValue); err != nil {
		slog.Error(fmt.Sprintf("while inserting action %q with message %s for instance ID %s: %v", actionType, message, instanceID, err))
	}
}

func (a *Action) ListActionsByInstanceID(instanceID string) ([]internal.Action, error) {
	return a.Factory.NewReadSession().ListActions(instanceID)
}
