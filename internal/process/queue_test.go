package process

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type StdExecutor struct {
	logger func(string)
}

func (e *StdExecutor) Execute(operationID string) (time.Duration, error) {
	e.logger(fmt.Sprintf("executing operation %s", operationID))
	return 0, nil
}

func gaugeValue(t *testing.T, queueName string) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, queueDepthMetric.WithLabelValues(queueName).Write(&m))
	return m.GetGauge().GetValue()
}

func TestQueueDepthMetric(t *testing.T) {
	name := fmt.Sprintf("depth-test-%d", time.Now().UnixNano())
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	processing := make(chan struct{})
	release := make(chan struct{})

	q := NewQueue(&StdExecutor{logger: func(_ string) {
		close(processing)
		<-release
	}}, logger, name)

	assert.Equal(t, 0.0, gaugeValue(t, name), "depth should be 0 before any items added")

	q.Add("op-1")
	assert.Equal(t, 1.0, gaugeValue(t, name), "depth should be 1 after Add")

	ctx, cancel := context.WithCancel(context.Background())
	q.Run(ctx.Done(), 1)

	// wait until the worker picks up the item
	<-processing
	assert.Equal(t, 0.0, gaugeValue(t, name), "depth should be 0 after Get")

	close(release)
	cancel()
	q.ShutDown()
	q.waitGroup.Wait()
}

func TestWorkerLogging(t *testing.T) {

	t.Run("should not log duplicated operationID", func(t *testing.T) {
		// given
		cw := &captureWriter{buf: &bytes.Buffer{}}
		handler := slog.NewTextHandler(cw, nil)
		logger := slog.New(handler)

		cancelContext, cancel := context.WithCancel(context.Background())
		var waitForProcessing sync.WaitGroup

		queue := NewQueue(&StdExecutor{logger: func(msg string) {
			t.Log(msg)
			waitForProcessing.Done()
		}}, logger, "test")

		waitForProcessing.Add(2)
		queue.AddAfter("processId2", 0)
		queue.Add("processId")
		queue.SpeedUp(1)
		queue.Run(cancelContext.Done(), 1)

		waitForProcessing.Wait()

		queue.ShutDown()
		cancel()
		queue.waitGroup.Wait()

		// then
		stringLogs := cw.buf.String()
		t.Log(stringLogs)
		require.NotContains(t, stringLogs, "operationID=processId2 operationID=processId")
	})

}

type captureWriter struct {
	buf *bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}
