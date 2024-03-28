package subaccountsync

type EventWindow struct {
	lastFromTime  int64
	lastToTime    int64
	windowSize    int64
	nowMillisFunc func() int64
}

func NewEventWindow(windowSize int64, nowFunc func() int64) *EventWindow {
	return &EventWindow{
		nowMillisFunc: nowFunc,
		windowSize:    windowSize,
	}
}

func (ew *EventWindow) GetNextFromTime() int64 {
	eventsFrom := ew.nowMillisFunc() - ew.windowSize
	if eventsFrom > ew.lastToTime {
		eventsFrom = ew.lastToTime
	}
	if eventsFrom < 0 {
		eventsFrom = 0
	}
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
