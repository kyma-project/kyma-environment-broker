package subaccountsync

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"log/slog"
)

type EventWindow struct {
	lastFromTime  int64
	lastToTime    int64
	windowSize    int64
	nowMillisFunc func() int64
	log           slog.Logger
}

func NewEventWindow(windowSize int64, log *slog.Logger, nowFunc func() int64) *EventWindow {
	return &EventWindow{
		nowMillisFunc: nowFunc,
		windowSize:    windowSize,
		log:           *log,
	}
}

func (ew *EventWindow) GetNextFromTime() int64 {
	epochNow := ew.nowMillisFunc()
	eventsFrom := epochNow - ew.windowSize
	if eventsFrom > ew.lastToTime {
		eventsFrom = ew.lastToTime
	}
	if eventsFrom < 0 {
		eventsFrom = 0
	}
	log.Info(fmt.Sprintf("Now: %d, From: %d, LastTo:% d", epochNow, eventsFrom, ew.lastToTime))
	return eventsFrom
}

func (ew *EventWindow) UpdateFromTime(fromTime int64) {
	ew.lastFromTime = fromTime
}

func (ew *EventWindow) UpdateToTime(eventTime int64) {
	if eventTime > ew.lastToTime {
		ew.lastToTime = eventTime
	}
}
