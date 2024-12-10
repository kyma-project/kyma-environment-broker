package broker

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

type Poller interface {
	Invoke(logic func() (bool, error)) error
}

type DefaultPoller struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
}

func NewDefaultPoller() Poller {
	return &DefaultPoller{
		PollInterval: 2 * time.Second,
		PollTimeout:  5 * time.Second,
	}
}

func (p *DefaultPoller) Invoke(logic func() (bool, error)) error {
	return wait.PollUntilContextTimeout(context.Background(), p.PollInterval, p.PollTimeout, true, func(ctx context.Context) (bool, error) {
		return logic()
	})
}

type PassthroughPoller struct {
}

func NewPassthroughPoller() Poller {
	return &PassthroughPoller{}
}

func (p *PassthroughPoller) Invoke(logic func() (bool, error)) error {
	success, err := logic()
	if !success && err == nil {
		return fmt.Errorf("unsuccessful poll logic invocation")
	}
	return err
}

type TimerPoller struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
	Log          func(args ...any)
}

func (p *TimerPoller) Invoke(logic func() (bool, error)) error {
	var start = time.Now()
	result := wait.PollUntilContextTimeout(context.Background(), p.PollInterval, p.PollTimeout, true, func(ctx context.Context) (bool, error) {
		return logic()
	})
	p.Log(fmt.Sprintf("Waiting for the logic execution took: %v", time.Since(start)))
	return result
}
