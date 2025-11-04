package process

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type StdExecutor struct {
	logger func(string)
}

func (e *StdExecutor) Execute(operationID string) (time.Duration, error) {
	e.logger(fmt.Sprintf("executing operation %s", operationID))
	return 0, nil
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
