package event_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestPubSub(t *testing.T) {
	// given
	var gotEventAList1 []eventA
	var gotEventAList2 []eventA
	var mu sync.Mutex
	handlerA1 := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		gotEventAList1 = append(gotEventAList1, ev.(eventA))
		return nil
	}
	handlerA2 := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		gotEventAList2 = append(gotEventAList2, ev.(eventA))
		return nil
	}
	var gotEventBList []eventB
	handlerB := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		gotEventBList = append(gotEventBList, ev.(eventB))
		return nil
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	svc := event.NewPubSub(log)
	svc.Subscribe(eventA{}, handlerA1)
	svc.Subscribe(eventB{}, handlerB)
	svc.Subscribe(eventA{}, handlerA2)

	// when
	svc.Publish(context.TODO(), eventA{msg: "first event"})
	svc.Publish(context.TODO(), eventB{msg: "second event"})
	svc.Publish(context.TODO(), eventA{msg: "third event"})

	time.Sleep(1 * time.Millisecond)

	// then
	assert.NoError(t, wait.PollImmediate(20*time.Millisecond, 2*time.Second, func() (bool, error) {
		return containsA(gotEventAList1, eventA{msg: "first event"}) &&
			containsA(gotEventAList1, eventA{msg: "third event"}) &&
			containsA(gotEventAList2, eventA{msg: "first event"}) &&
			containsA(gotEventAList2, eventA{msg: "third event"}) &&
			containsB(gotEventBList, eventB{msg: "second event"}), nil
	}))
}

func TestPubSub_WhenHandlerReturnsError(t *testing.T) {
	// given
	cw := &captureWriter{entries: []string{}}
	handler := slog.NewTextHandler(cw, nil)
	log := slog.New(handler)
	var mu sync.Mutex
	handlerA1 := func(ctx context.Context, ev interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		return fmt.Errorf("some error")
	}
	svc := event.NewPubSub(log)
	svc.Subscribe(eventA{}, handlerA1)

	// when
	svc.Publish(context.TODO(), eventA{msg: "first event"})

	time.Sleep(1 * time.Millisecond)

	// then
	require.Equal(t, 1, len(cw.entries))
	require.Contains(t, cw.entries[0], "error while calling pubsub event handler: some error")
}

func containsA(slice []eventA, item eventA) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsB(slice []eventB, item eventB) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type eventA struct {
	msg string
}

type eventB struct {
	msg string
}

type captureWriter struct {
	entries []string
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	entry := string(p)
	c.entries = append(c.entries, entry)
	return len(p), nil
}
