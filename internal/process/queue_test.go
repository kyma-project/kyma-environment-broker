package process

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
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

	t.Run("should log basic worker information", func(t *testing.T) {
		// given
		logger := logrus.New()

		var logs bytes.Buffer
		logger.SetOutput(&logs)

		cancelContext, cancel := context.WithCancel(context.Background())
		var waitForProcessing sync.WaitGroup

		queue := NewQueue(&StdExecutor{logger: func(msg string) {
			t.Log(msg)
			waitForProcessing.Done()
		}}, logger, "test", 10*time.Millisecond, 10*time.Millisecond)

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
		stringLogs := logs.String()
		t.Log(stringLogs)
		require.True(t, strings.Contains(stringLogs, "msg=\"starting 1 worker(s), queue length is 2\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"starting worker with id 0\" queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"item processId2 will be added to the queue test after duration of 0, queue length is 1\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"added item processId to the queue test, queue length is 2\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"processing item processId2, queue length is 1\" operationID=processId2 queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"processing item processId, queue length is 0\" operationID=processId queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"shutting down the queue, queue length is 0\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"queue speed factor set to 1\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"worker routine - starting\" queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"worker done\" queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"shutting down\" operationID=processId queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"item for processId has been processed, no retry, element forgotten\" operationID=processId queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"about to process item processId, queue length is 0\" operationID=processId queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"execution - worker test-0 last execution time"))
		require.True(t, strings.Contains(stringLogs, "executed after"))
	})

}
