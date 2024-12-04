package events

import (
	"fmt"
	"log/slog"
	"time"

	eventsapi "github.com/kyma-project/kyma-environment-broker/common/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
)

type events struct {
	postsql.Factory
}

func New(fac postsql.Factory) *events {
	return &events{Factory: fac}
}

func (e *events) ListEvents(filter eventsapi.EventFilter) ([]eventsapi.EventDTO, error) {
	if e == nil {
		return nil, fmt.Errorf("events are disabled")
	}
	sess := e.NewReadSession()
	return sess.ListEvents(filter)
}

func (e *events) InsertEvent(eventLevel eventsapi.EventLevel, message, instanceID, operationID string) {
	if e == nil {
		return
	}
	sess := e.NewWriteSession()
	if err := sess.InsertEvent(eventLevel, message, instanceID, operationID); err != nil {
		slog.Error(fmt.Sprintf("failed to insert event [%v] %v/%v %q", eventLevel, instanceID, operationID, message))
	}
}

func (e *events) RunGarbageCollection(pollingPeriod, retention time.Duration) {
	if e == nil {
		return
	}
	if retention == 0 {
		return
	}
	ticker := time.NewTicker(pollingPeriod)
	for {
		select {
		case <-ticker.C:
			sess := e.NewWriteSession()
			if err := sess.DeleteEvents(time.Now().Add(-retention)); err != nil {
				slog.Error(fmt.Sprintf("failed to delete old events: %v", err))
			}
		}
	}
}
