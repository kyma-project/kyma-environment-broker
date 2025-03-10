package event

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

type Handler = func(ctx context.Context, ev interface{}) error

type Publisher interface {
	Publish(ctx context.Context, event interface{})
}

type Subscriber interface {
	Subscribe(evType interface{}, evHandler Handler)
}

// PubSub implements a simple event broker which allows to send event across the application.
type PubSub struct {
	mu  sync.Mutex
	log *slog.Logger

	handlers map[reflect.Type][]Handler
}

func NewPubSub(log *slog.Logger) *PubSub {
	return &PubSub{
		log:      log,
		handlers: make(map[reflect.Type][]Handler),
	}
}

func (b *PubSub) Publish(ctx context.Context, ev interface{}) {
	tt := reflect.TypeOf(ev)
	hList, found := b.handlers[tt]
	if found {
		for _, handler := range hList {
			go func(h Handler) {
				err := h(ctx, ev)
				if err != nil {
					b.log.Error(fmt.Sprintf("error while calling pubsub event handler: %s", err.Error()))
				}
			}(handler)
		}
	}
}

func (b *PubSub) Subscribe(evType interface{}, evHandler Handler) {
	tt := reflect.TypeOf(evType)
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, found := b.handlers[tt]; !found {
		b.handlers[tt] = []Handler{}
	}

	b.handlers[tt] = append(b.handlers[tt], evHandler)
}
