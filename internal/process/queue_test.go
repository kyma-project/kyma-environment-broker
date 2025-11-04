package process

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
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

	t.Run("should log basic worker information", func(t *testing.T) {
		// given
		cw := &captureWriter{buf: &bytes.Buffer{}}
		handler := slog.NewTextHandler(cw, nil)
		logger := slog.New(handler)

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
		stringLogs := cw.buf.String()
		t.Log(stringLogs)
		require.True(t, strings.Contains(stringLogs, "msg=\"item processId2 will be added to the queue test after duration of 0, queue length is 1\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"added item processId to the queue test, queue length is 2\" queueName=test"))

		require.True(t, strings.Contains(stringLogs, "msg=\"updating worker time, processing item processId2, queue length is 1\" queueName=test workerId=0 operationID=processId2"))
		require.True(t, strings.Contains(stringLogs, "msg=\"updating worker time, processing item processId, queue length is 0\" queueName=test workerId=0 operationID=processId"))
		require.True(t, strings.Contains(stringLogs, "msg=\"shutting down the queue, queue length is 0\" queueName=test"))
		require.True(t, strings.Contains(stringLogs, "msg=\"queue speed factor set to 1\" queueName=test"))

		require.True(t, strings.Contains(stringLogs, "msg=\"shutting down\" queueName=test workerId=0"))
		require.True(t, strings.Contains(stringLogs, "msg=\"item for processId has been processed, no retry, element forgotten\" queueName=test workerId=0 operationID=processId"))
		require.True(t, strings.Contains(stringLogs, "msg=\"about to process item processId, queue length is 0\" queueName=test workerId=0 operationID=processId"))
	})

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
		stringLogs := cw.buf.String()
		t.Log(stringLogs)
		require.NotContains(t, stringLogs, "operationID=processId2 operationID=processId")
	})

}

func TestQueueMutexProtection(t *testing.T) {
	t.Run("concurrent map access should not cause race condition and panic", func(t *testing.T) {
		const numGoroutines = 50
		const numOperations = 100
		var wg sync.WaitGroup

		cw := &captureWriter{buf: &bytes.Buffer{}}
		handler := slog.NewTextHandler(cw, nil)
		logger := slog.New(handler)
		queue := NewQueue(&StdExecutor{logger: func(msg string) {}}, logger, "mutex-test", 10*time.Millisecond, 10*time.Millisecond)

		testConcurrentAccess := func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < numOperations; i++ {
				workerID := fmt.Sprintf("worker-%d", goroutineID)
				key := fmt.Sprintf("key-%d-%d", goroutineID, i)

				// 1. updateWorkerTime writes to the maps
				queue.updateWorkerTime(key, workerID, logger)

				// 2. logWorkersSummary reads from the maps
				if i%10 == 0 {
					queue.logWorkersSummary()
				}

				// 3. removeTimeIfInWarnMargin reads then deletes from maps
				queue.removeTimeIfInWarnMargin(key, workerID, logger)
			}
		}

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go testConcurrentAccess(i)
		}
		wg.Wait()

		// Verify that all worker data has been properly cleaned up
		queue.workerMutex.RLock()
		executionTimesCount := len(queue.workerExecutionTimes)
		lastKeysCount := len(queue.workerLastKeys)
		queue.workerMutex.RUnlock()
		require.Equal(t, 0, executionTimesCount, "All worker execution times should be cleaned up")
		require.Equal(t, 0, lastKeysCount, "All worker last keys should be cleaned up")
	})
}

type captureWriter struct {
	buf *bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}
